package database

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var ErrUnsupportedDataQuestion = errors.New("question cannot be mapped safely to the available business data")

func IsSchemaQuestion(question string) bool {
	normalized := strings.ToLower(question)
	return containsAny(normalized, "which table", "what table", "what column", "which column",
		"primary key", "foreign key", "information_schema", "window function", "query plan",
		"how do i join", "how are tables", "how are .* related")
}

type DataAssistant struct {
	config  Config
	db      *sql.DB
	catalog Catalog
}

type DataAnswer struct {
	Answer     string
	Query      string
	Columns    []string
	Rows       [][]string
	Confidence float64
}

func NewDataAssistant(ctx context.Context, config Config) (*DataAssistant, error) {
	if config.Provider != ProviderMySQL && config.Provider != ProviderPostgreSQL {
		return nil, fmt.Errorf("business data queries currently support MySQL and PostgreSQL")
	}
	if !config.ReadOnly {
		return nil, fmt.Errorf("business data queries require NYANSAPO_DB_READ_ONLY=true")
	}
	driver, dsn, err := connectionDriverAndDSN(config)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	catalog, err := IntrospectSQL(ctx, db, config.Provider, config.Database, config.Name)
	if err != nil {
		db.Close()
		return nil, err
	}
	return &DataAssistant{config: config, db: db, catalog: catalog}, nil
}

func (a *DataAssistant) Close() error {
	if a == nil || a.db == nil {
		return nil
	}
	return a.db.Close()
}

func (a *DataAssistant) Ask(ctx context.Context, question string) (DataAnswer, error) {
	query, err := a.Plan(question)
	if err != nil {
		return DataAnswer{}, err
	}
	if err := ValidateReadOnlyQuery(query); err != nil {
		return DataAnswer{}, fmt.Errorf("generated query rejected: %w", err)
	}
	queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	rows, err := a.db.QueryContext(queryCtx, query)
	if err != nil {
		return DataAnswer{}, fmt.Errorf("run business data query: %w", err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return DataAnswer{}, fmt.Errorf("read query columns: %w", err)
	}
	values := make([][]string, 0, 100)
	for rows.Next() {
		raw := make([]any, len(columns))
		destinations := make([]any, len(columns))
		for index := range raw {
			destinations[index] = &raw[index]
		}
		if err := rows.Scan(destinations...); err != nil {
			return DataAnswer{}, fmt.Errorf("read query result: %w", err)
		}
		record := make([]string, len(raw))
		for index, value := range raw {
			record[index] = formatValue(value)
		}
		values = append(values, record)
		if len(values) == 100 {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return DataAnswer{}, fmt.Errorf("read query results: %w", err)
	}
	return DataAnswer{
		Answer:     formatAnswer(question, columns, values),
		Query:      query,
		Columns:    columns,
		Rows:       values,
		Confidence: 1,
	}, nil
}

func (a *DataAssistant) Plan(question string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(question))
	if normalized == "" {
		return "", ErrUnsupportedDataQuestion
	}
	switch {
	case containsAny(normalized, "absent", "absence", "attendance"):
		for _, table := range rankedTables(a.catalog.Tables, []string{"absence", "absent", "attendance", "leave"}) {
			if query, ok := absenceQuery(a.config.Provider, table); ok {
				return query, nil
			}
		}
	case containsAny(normalized, "how many", "count", "number of"):
		if table, ok := bestTable(a.catalog.Tables, questionWords(normalized)); ok {
			return countQuery(a.config.Provider, table), nil
		}
	case containsAny(normalized, "money", "revenue", "income", "sales", "payment", "paid", "amount"):
		for _, table := range rankedTables(a.catalog.Tables, []string{"payment", "transaction", "sale", "invoice", "revenue", "income", "order"}) {
			if query, ok := revenueQuery(a.config.Provider, table, normalized); ok {
				return query, nil
			}
		}
	}
	return "", ErrUnsupportedDataQuestion
}

func absenceQuery(provider Provider, table Table) (string, bool) {
	name := findColumn(table, []string{"staff_name", "employee_name", "full_name", "name", "staff_id", "employee_id", "user_id", "person_id"})
	date := findColumn(table, []string{"attendance_date", "absence_date", "absent_date", "date", "day", "occurred_at", "created_at"})
	status := findColumn(table, []string{"status", "attendance_status", "state", "reason"})
	if name == "" || date == "" {
		return "", false
	}
	columns := []string{quoteIdentifier(provider, name)}
	if date != "" {
		columns = append(columns, quoteIdentifier(provider, date))
	}
	if status != "" {
		columns = append(columns, quoteIdentifier(provider, status))
	}
	tableName := qualifiedTable(provider, table)
	dateFilter := dateExpression(provider, date)
	statusFilter := ""
	if status != "" {
		statusFilter = fmt.Sprintf(" AND LOWER(CAST(%s AS CHAR)) IN ('absent', 'absence', 'sick', 'leave')", quoteIdentifier(provider, status))
		if provider == ProviderPostgreSQL {
			statusFilter = fmt.Sprintf(" AND LOWER(CAST(%s AS TEXT)) IN ('absent', 'absence', 'sick', 'leave')", quoteIdentifier(provider, status))
		}
	}
	return fmt.Sprintf("SELECT %s FROM %s WHERE %s >= %s%s ORDER BY %s DESC LIMIT 100",
		strings.Join(columns, ", "), tableName, quoteIdentifier(provider, date), dateFilter, statusFilter, quoteIdentifier(provider, date)), true
}

func revenueQuery(provider Provider, table Table, question string) (string, bool) {
	amount := findColumn(table, []string{"amount", "total_amount", "paid_amount", "net_amount", "total", "revenue", "price", "value"})
	date := findColumn(table, []string{"payment_date", "transaction_date", "sale_date", "invoice_date", "date", "created_at", "paid_at"})
	if amount == "" || date == "" {
		return "", false
	}
	sum := fmt.Sprintf("SUM(%s)", quoteIdentifier(provider, amount))
	return fmt.Sprintf("SELECT %s AS total_%s FROM %s WHERE %s >= %s",
		sum, amount, qualifiedTable(provider, table), quoteIdentifier(provider, date), dateExpression(provider, date)), true
}

func countQuery(provider Provider, table Table) string {
	return fmt.Sprintf("SELECT COUNT(*) AS total FROM %s", qualifiedTable(provider, table))
}

func dateExpression(provider Provider, column string) string {
	if strings.Contains(column, "created") || strings.HasSuffix(column, "_at") {
		if provider == ProviderMySQL {
			return "CURRENT_TIMESTAMP - INTERVAL 7 DAY"
		}
		return "CURRENT_TIMESTAMP - INTERVAL '7 days'"
	}
	if provider == ProviderMySQL {
		return "CURRENT_DATE - INTERVAL 7 DAY"
	}
	return "CURRENT_DATE - INTERVAL '7 days'"
}

func bestTable(tables []Table, terms []string) (Table, bool) {
	ranked := rankedTables(tables, terms)
	if len(ranked) == 0 {
		return Table{}, false
	}
	return ranked[0], true
}

func rankedTables(tables []Table, terms []string) []Table {
	type scored struct {
		table Table
		score int
	}
	ranked := make([]scored, 0, len(tables))
	for _, table := range tables {
		score := 0
		tableName := strings.ToLower(table.Name)
		for _, term := range terms {
			term = strings.TrimSuffix(strings.ToLower(term), "s")
			if term == "" {
				continue
			}
			if strings.Contains(tableName, term) {
				score += 4
			}
			for _, column := range table.Columns {
				if strings.Contains(strings.ToLower(column.Name), term) {
					score++
				}
			}
		}
		if score > 0 {
			ranked = append(ranked, scored{table: table, score: score})
		}
	}
	for i := 1; i < len(ranked); i++ {
		for j := i; j > 0 && ranked[j].score > ranked[j-1].score; j-- {
			ranked[j], ranked[j-1] = ranked[j-1], ranked[j]
		}
	}
	tablesRanked := make([]Table, 0, len(ranked))
	for _, item := range ranked {
		tablesRanked = append(tablesRanked, item.table)
	}
	return tablesRanked
}

func findColumn(table Table, names []string) string {
	for _, name := range names {
		for _, column := range table.Columns {
			if strings.EqualFold(column.Name, name) {
				return column.Name
			}
		}
	}
	for _, column := range table.Columns {
		lower := strings.ToLower(column.Name)
		for _, name := range names {
			if strings.Contains(lower, strings.ToLower(name)) {
				return column.Name
			}
		}
	}
	return ""
}

func qualifiedTable(provider Provider, table Table) string {
	if table.Schema == "" {
		return quoteIdentifier(provider, table.Name)
	}
	return quoteIdentifier(provider, table.Schema) + "." + quoteIdentifier(provider, table.Name)
}

func quoteIdentifier(provider Provider, identifier string) string {
	if provider == ProviderMySQL {
		return "`" + strings.ReplaceAll(identifier, "`", "``") + "`"
	}
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

func questionWords(question string) []string {
	return strings.FieldsFunc(question, func(r rune) bool {
		return r < 'a' || r > 'z'
	})
}

func containsAny(value string, terms ...string) bool {
	for _, term := range terms {
		if strings.Contains(value, term) {
			return true
		}
	}
	return false
}

func formatValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return "NULL"
	case []byte:
		return string(typed)
	case time.Time:
		return typed.Format(time.RFC3339)
	case int64:
		return strconv.FormatInt(typed, 10)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 2)
	default:
		return fmt.Sprint(typed)
	}
}

func formatAnswer(question string, columns []string, rows [][]string) string {
	if len(rows) == 0 {
		return pick(noRecordTemplates)
	}
	normalized := strings.ToLower(question)
	nameIndex := indexOfColumn(columns, "employee", "staff", "name")
	dateIndex := indexOfColumn(columns, "date", "created", "paid", "occurred")
	statusIndex := indexOfColumn(columns, "status", "state", "reason")

	if len(rows) == 1 && len(columns) == 1 {
		value := rows[0][0]
		columnLower := strings.ToLower(columns[0])
		switch {
		case containsAny(normalized, "absent", "absence", "attendance"):
			return strings.ReplaceAll(pick(absentCountTemplates), "{count}", formattedCount(value))
		case columnLower == "total" || containsAny(normalized, "how many", "count", "number of"):
			return strings.ReplaceAll(pick(countTemplates), "{count}", value)
		case strings.HasPrefix(columnLower, "total_") || containsAny(normalized, "money", "revenue", "income", "sales", "payment", "paid", "amount"):
			return strings.ReplaceAll(pick(revenueTemplates), "{amount}", value)
		default:
			return fmt.Sprintf("The result is %s.", value)
		}
	}

	if dateIndex >= 0 && statusIndex >= 0 && nameIndex >= 0 {
		sentences := make([]string, 0, len(rows))
		for _, row := range rows {
			person := displayPerson(row[nameIndex], columns[nameIndex])
			status := row[statusIndex]
			date := displayDate(row[dateIndex])
			sentences = append(sentences, pickAttendanceRow(person, status, date))
		}
		body := strings.Join(sentences, " ")
		if len(sentences) > 1 {
			return strings.ReplaceAll(pick(attendanceIntroTemplates), "{body}", body)
		}
		return body
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Found %d matching record(s).\n", len(rows)))
	builder.WriteString(strings.Join(columns, " | "))
	builder.WriteByte('\n')
	for _, row := range rows {
		builder.WriteString(strings.Join(row, " | "))
		builder.WriteByte('\n')
	}
	return strings.TrimSpace(builder.String())
}

var noRecordTemplates = []string{
	"No matching records were found in the last seven days.",
	"I could not find any matching records in the last seven days.",
	"There are no matching records for the last seven days.",
	"Nothing matched in the last seven days.",
}

var absentCountTemplates = []string{
	"{count} attendance record(s) matched for the last seven days.",
	"I found {count} attendance record(s) for the last seven days.",
	"{count} attendance record(s) matched in the seven-day window.",
	"The last seven days show {count} matching attendance record(s).",
}

var countTemplates = []string{
	"There are {count} in total.",
	"In total, there are {count}.",
	"The total count is {count}.",
	"We have {count} records in total.",
}

var revenueTemplates = []string{
	"We received {amount} in the last seven days.",
	"The total received in the last seven days is {amount}.",
	"In the last seven days we took in {amount}.",
	"{amount} came in over the last seven days.",
	"Our receipts for the last seven days totaled {amount}.",
}

var attendanceIntroTemplates = []string{
	"From the attendance records: {body}",
	"According to the records: {body}",
	"Attendance shows: {body}",
	"Here is what the attendance data shows: {body}",
	"Reviewing the attendance records: {body}",
}

var attendanceRowTemplates = []string{
	"{person} was {status} on {date}.",
	"{person} was marked {status} on {date}.",
	"On {date}, {person} was {status}.",
	"{person} was recorded as {status} on {date}.",
	"Records show {person} {status} on {date}.",
}

func pickAttendanceRow(person, status, date string) string {
	template := pick(attendanceRowTemplates)
	template = strings.ReplaceAll(template, "{person}", person)
	template = strings.ReplaceAll(template, "{status}", status)
	template = strings.ReplaceAll(template, "{date}", date)
	return template
}

func pick(templates []string) string {
	return templates[randomIndex(len(templates))]
}

func randomIndex(count int) int {
	if count < 2 {
		return 0
	}
	var value [1]byte
	if _, err := rand.Read(value[:]); err == nil {
		return int(value[0]) % count
	}
	return 0
}

func indexOfColumn(columns []string, needles ...string) int {
	for index, column := range columns {
		lower := strings.ToLower(column)
		for _, needle := range needles {
			if strings.Contains(lower, needle) {
				return index
			}
		}
	}
	return -1
}

func displayPerson(value, column string) string {
	lower := strings.ToLower(column)
	if strings.Contains(lower, "name") {
		return value
	}
	if strings.Contains(lower, "id") {
		return "Employee " + value
	}
	return value
}

func displayDate(value string) string {
	if formatted, err := time.Parse(time.RFC3339, value); err == nil {
		return formatted.Format("January 2, 2006")
	}
	return value
}

func formattedCount(value string) string {
	if value == "1" || value == "1.0" {
		return "One"
	}
	if value == "2" || value == "2.0" {
		return "Two"
	}
	return value
}
