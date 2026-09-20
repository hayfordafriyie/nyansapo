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
	if !strings.Contains(got, "Photosynthesis converts light energy") {
		t.Fatalf("Format() = %q, want photosynthesis sentence", got)
	}
	if strings.Contains(got, "Science studies nature") {
		t.Fatalf("Format() returned unrelated sentence: %q", got)
	}
}

func TestFormatKeepsFactsForSameQuestion(t *testing.T) {
	passage := "Photosynthesis converts light energy into chemical energy. Chlorophyll absorbs light."
	for range 10 {
		answer := Format("Explain photosynthesis", passage)
		if !strings.Contains(answer, "Photosynthesis converts light energy") ||
			!strings.Contains(answer, "Chlorophyll absorbs light") {
			t.Fatalf("answer lost source facts: %q", answer)
		}
	}
}

func TestFormatBuildsExplanationFromRelatedContext(t *testing.T) {
	got := Format(
		"What is photosynthesis?",
		"Science studies nature. Photosynthesis converts light energy into chemical energy. Chlorophyll absorbs light.",
	)
	if !strings.Contains(got, "Photosynthesis converts light energy") ||
		!strings.Contains(got, "Chlorophyll absorbs light") {
		t.Fatalf("Format() = %q, want explanation with related context", got)
	}
}
