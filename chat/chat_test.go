package chat

import "testing"

func TestAnswer(t *testing.T) {
	knowledge := Knowledge{
		Description: "EDSPiKE is a school management and learning platform.",
		Country:     "Ghana",
		Features:    []string{"Student management", "Teacher management"},
	}

	tests := []struct {
		question string
		want     string
	}{
		{"What is EDSPiKE?", knowledge.Description},
		{"Which country?", knowledge.Country},
		{"List features", "Student management, Teacher management"},
		{"Who are you?", "I do not know that yet."},
	}

	for _, test := range tests {
		if got := Answer(test.question, knowledge); got != test.want {
			t.Errorf("Answer(%q) = %q, want %q", test.question, got, test.want)
		}
	}
}
