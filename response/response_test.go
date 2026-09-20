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

func TestFormatIsStableForSameQuestion(t *testing.T) {
	passage := "Photosynthesis converts light energy into chemical energy. Chlorophyll absorbs light."
	if first, second := Format("Explain photosynthesis", passage), Format("Explain photosynthesis", passage); first != second {
		t.Fatalf("same question produced different answers: %q and %q", first, second)
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
