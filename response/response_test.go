package response

import (
	"strings"
	"testing"
)

func TestFormatSelectsRelevantSentence(t *testing.T) {
	got := Format(
		"What is photosynthesis?",
		"Science studies nature. Photosynthesis converts light energy into chemical energy. Chlorophyll absorbs light.",
	)
	if !strings.Contains(strings.ToLower(got), "photosynthesis") ||
		!strings.Contains(strings.ToLower(got), "chemical energy") {
		t.Fatalf("Format() = %q, want photosynthesis sentence", got)
	}
	if strings.Contains(got, "Science studies nature") {
		t.Fatalf("Format() returned unrelated sentence: %q", got)
	}
}

func TestFormatKeepsFactsForSameQuestion(t *testing.T) {
	passage := "Photosynthesis converts light energy into chemical energy. Chlorophyll absorbs light."
	answers := make(map[string]struct{})
	for range 20 {
		answer := Format("Explain photosynthesis", passage)
		if !strings.Contains(strings.ToLower(answer), "photosynthesis") ||
			!strings.Contains(strings.ToLower(answer), "chlorophyll") {
			t.Fatalf("answer lost source facts: %q", answer)
		}
		answers[answer] = struct{}{}
	}
	if len(answers) < 2 {
		t.Fatalf("expected paraphrase variation, got %d answer", len(answers))
	}
}

func TestFormatBuildsExplanationFromRelatedContext(t *testing.T) {
	got := Format(
		"What is photosynthesis?",
		"Science studies nature. Photosynthesis converts light energy into chemical energy. Chlorophyll absorbs light.",
	)
	if !strings.Contains(strings.ToLower(got), "photosynthesis") ||
		!strings.Contains(strings.ToLower(got), "chlorophyll") {
		t.Fatalf("Format() = %q, want explanation with related context", got)
	}
}
