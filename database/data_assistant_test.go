package database

import (
	"strings"
	"testing"
)

func testBusinessCatalog() Catalog {
	return Catalog{
		Provider: ProviderPostgreSQL,
		Tables: []Table{
			{
				Schema: "public",
				Name:   "staff_attendance",
				Columns: []Column{
					{Name: "staff_id", Type: "integer"},
					{Name: "attendance_date", Type: "date"},
					{Name: "status", Type: "text"},
				},
			},
			{
				Schema: "public",
				Name:   "payments",
				Columns: []Column{
					{Name: "amount", Type: "numeric"},
					{Name: "paid_at", Type: "timestamp"},
				},
			},
		},
	}
}

func TestPlanAbsenceQuestionUsesBusinessData(t *testing.T) {
	assistant := &DataAssistant{config: Config{Provider: ProviderPostgreSQL}, catalog: testBusinessCatalog()}
	query, err := assistant.Plan("Which staff has been absent for a week?")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(query, `"staff_attendance"`) ||
		!strings.Contains(query, `"attendance_date"`) ||
		!strings.Contains(query, "INTERVAL '7 days'") {
		t.Fatalf("query = %q, want bounded attendance query", query)
	}
	if err := ValidateReadOnlyQuery(query); err != nil {
		t.Fatalf("generated query is not read-only: %v", err)
	}
}

func TestPlanRevenueQuestionUsesBusinessData(t *testing.T) {
	assistant := &DataAssistant{config: Config{Provider: ProviderPostgreSQL}, catalog: testBusinessCatalog()}
	query, err := assistant.Plan("How much money did we get in the last week?")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(query, "SUM") ||
		!strings.Contains(query, `"amount"`) ||
		!strings.Contains(query, `"paid_at"`) {
		t.Fatalf("query = %q, want revenue aggregate", query)
	}
}

func TestPlanRejectsSchemaOnlyQuestion(t *testing.T) {
	assistant := &DataAssistant{config: Config{Provider: ProviderPostgreSQL}, catalog: testBusinessCatalog()}
	if _, err := assistant.Plan("Which tables contain customer orders?"); err == nil {
		t.Fatal("schema-only question was accepted as a business data query")
	}
}

func TestPlanCountQuestionWithSynonyms(t *testing.T) {
	catalog := Catalog{
		Provider: ProviderPostgreSQL,
		Tables: []Table{
			{Schema: "public", Name: "employees", Columns: []Column{{Name: "id", Type: "integer"}}},
			{Schema: "public", Name: "customers", Columns: []Column{{Name: "id", Type: "integer"}}},
		},
	}
	assistant := &DataAssistant{config: Config{Provider: ProviderPostgreSQL}, catalog: catalog}

	query, err := assistant.Plan("How many staff do we have?")
	if err != nil {
		t.Fatalf("staff synonym should map to employees: %v", err)
	}
	if !strings.Contains(query, `COUNT(*)`) || !strings.Contains(query, `"employees"`) {
		t.Fatalf("query = %q, want COUNT(*) on employees", query)
	}

	query, err = assistant.Plan("How many clients do we have?")
	if err != nil {
		t.Fatalf("clients synonym should map to customers: %v", err)
	}
	if !strings.Contains(query, `"customers"`) {
		t.Fatalf("query = %q, want COUNT(*) on customers", query)
	}
}

func TestFormatAnswerKeepsCoreFactsInSentences(t *testing.T) {
	answer := formatAnswer("How much money did we get in the last week?", []string{"total_amount"}, [][]string{{"26400.00"}})
	if !strings.Contains(answer, "26400.00") {
		t.Fatalf("answer lost the amount: %q", answer)
	}
	if strings.Contains(answer, "26400.00 |") || strings.Contains(answer, "\n") {
		t.Fatalf("answer is not a natural sentence: %q", answer)
	}
}

func TestFormatAnswerVariesWordingAcrossRuns(t *testing.T) {
	seen := make(map[string]struct{})
	for range 30 {
		answer := formatAnswer("How much money did we get in the last week?", []string{"total_amount"}, [][]string{{"26400.00"}})
		if !strings.Contains(answer, "26400.00") {
			t.Fatalf("answer lost the amount: %q", answer)
		}
		seen[answer] = struct{}{}
	}
	if len(seen) < 2 {
		t.Fatalf("expected varied sentence structures, got %d unique answer(s)", len(seen))
	}
}

func TestFormatAttendanceAnswerUsesSentences(t *testing.T) {
	answer := formatAnswer(
		"Which staff has been absent for a week?",
		[]string{"staff_id", "attendance_date", "status"},
		[][]string{{"1", "2026-09-19T00:00:00Z", "absent"}},
	)
	if !strings.Contains(answer, "absent") ||
		!strings.Contains(strings.ToLower(answer), "1") {
		t.Fatalf("answer = %q, want sentence with staff and status", answer)
	}
	if strings.Contains(answer, "|") {
		t.Fatalf("answer is a raw table, not a sentence: %q", answer)
	}
}

func TestColumnPickersDetectRolesFromTypesOnly(t *testing.T) {
	table := Table{Name: "feedback", Columns: []Column{
		{Name: "id", Type: "integer", PrimaryKey: true},
		{Name: "customer_name", Type: "character varying"},
		{Name: "rating", Type: "integer"},
		{Name: "submitted_on", Type: "timestamp"},
		{Name: "sentiment", Type: "text"},
	}}
	date, ok := pickTemporalColumn(table)
	if !ok || date.Name != "submitted_on" {
		t.Fatalf("temporal = %q, want submitted_on", date.Name)
	}
	skipped := map[string]bool{date.Name: true}
	amount, ok := pickAmountColumn(table, skipped)
	if !ok || amount.Name != "rating" {
		t.Fatalf("amount = %q, want rating", amount.Name)
	}
	person, ok := pickPersonColumn(table, skipped)
	if !ok || person.Name != "customer_name" {
		t.Fatalf("person = %q, want customer_name", person.Name)
	}
	skipped[person.Name] = true
	status, ok := pickStatusColumn(table, skipped)
	if !ok || status.Name != "sentiment" {
		t.Fatalf("status = %q, want sentiment", status.Name)
	}
}

func TestPlanDynamicSchemaWithoutKnownTableNames(t *testing.T) {
	catalog := Catalog{
		Provider: ProviderPostgreSQL,
		Tables: []Table{
			{
				Schema: "public",
				Name:   "feedback",
				Columns: []Column{
					{Name: "id", Type: "integer", PrimaryKey: true},
					{Name: "customer_name", Type: "text"},
					{Name: "rating", Type: "numeric"},
					{Name: "submitted_on", Type: "timestamp"},
				},
			},
		},
	}
	assistant := &DataAssistant{config: Config{Provider: ProviderPostgreSQL}, catalog: catalog}
	query, err := assistant.Plan("How much total rating did we get in the last week?")
	if err != nil {
		t.Fatalf("dynamic revenue query: %v", err)
	}
	if !strings.Contains(query, `"feedback"`) || !strings.Contains(query, "SUM(") {
		t.Fatalf("query = %q, want SUM over dynamic schema", query)
	}

	query, err = assistant.Plan("How many feedback records are there?")
	if err != nil {
		t.Fatalf("dynamic count query: %v", err)
	}
	if !strings.Contains(query, `"feedback"`) || !strings.Contains(query, "COUNT(*)") {
		t.Fatalf("query = %q, want COUNT over dynamic schema", query)
	}
}
