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
		!strings.Contains(strings.ToLower(answer), "employee 1") {
		t.Fatalf("answer = %q, want sentence with staff and status", answer)
	}
	if strings.Contains(answer, "|") {
		t.Fatalf("answer is a raw table, not a sentence: %q", answer)
	}
}
