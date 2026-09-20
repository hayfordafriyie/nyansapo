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

func TestFormatDoesNotAddUnrelatedNextSentence(t *testing.T) {
	got := Format(
		"What is a fraction?",
		"A fraction represents a part of a whole. The area of a rectangle is length multiplied by width.",
	)
	if strings.Contains(strings.ToLower(got), "rectangle") {
		t.Fatalf("Format() added unrelated context: %q", got)
	}
	if !strings.Contains(strings.ToLower(got), "fraction") {
		t.Fatalf("Format() lost the answer: %q", got)
	}
}

func TestFormatUsesNaturalSentenceCapitalization(t *testing.T) {
	got := Format(
		"What is photosynthesis?",
		"Photosynthesis allows plants to convert light energy into chemical energy.",
	)
	if strings.Contains(got, ": Photosynthesis") {
		return
	}
	if strings.Contains(got, ", Photosynthesis") || strings.Contains(got, "where Photosynthesis") {
		t.Fatalf("Format() produced awkward capitalization: %q", got)
	}
}

func TestFormatExplainsShortWhyAnswer(t *testing.T) {
	got := Format(
		"Why is chlorophyll important?",
		"Chlorophyll helps plants absorb light.",
	)
	if !strings.Contains(strings.ToLower(got), "because") &&
		!strings.Contains(strings.ToLower(got), "reason") &&
		!strings.Contains(strings.ToLower(got), "important") {
		t.Fatalf("Format() did not explain why: %q", got)
	}
}
