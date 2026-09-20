package conversation

import (
	"strings"
	"testing"
)

func TestEnrichWithoutContext(t *testing.T) {
	memory := NewMemory(0)
	if got := memory.Enrich("How many payments were received?"); got != "How many payments were received?" {
		t.Fatalf("Enrich() = %q, want unchanged", got)
	}
}

func TestEnrichShortReferenceUsesLastQuestion(t *testing.T) {
	memory := NewMemory(0)
	memory.Remember("How much money did we receive in the last week?", "We received 26400.00.")

	got := memory.Enrich("and how many payments were received?")
	if !strings.Contains(got, "How much money did we receive in the last week?") {
		t.Fatalf("Enrich() = %q, want last question appended", got)
	}
}

func TestEnrichReferenceWordsAppendContext(t *testing.T) {
	memory := NewMemory(0)
	memory.Remember("Which staff has been absent for a week?", "Employee 1 was absent.")

	var matched bool
	for _, question := range []string{"and the recent ones?", "same for payments", "what about employees?"} {
		if got := memory.Enrich(question); strings.Contains(got, "Which staff has been absent for a week?") {
			matched = true
		}
	}
	if !matched {
		t.Fatal("reference-word questions did not append context")
	}
}

func TestRememberKeepsOnlyRecentTurns(t *testing.T) {
	memory := NewMemory(2)
	memory.Remember("one?", "a")
	memory.Remember("two?", "b")
	memory.Remember("three?", "c")

	got := memory.Enrich("it again")
	if strings.Contains(got, "one?") {
		t.Fatalf("Enrich() = %q, old turn still present", got)
	}
	if !strings.Contains(got, "three?") {
		t.Fatalf("Enrich() = %q, latest turn missing", got)
	}
}

func TestRepeatReturnsLastAnswer(t *testing.T) {
	memory := NewMemory(0)
	memory.Remember("How much money did we receive in the last week?", "In the last seven days we took in 26400.00.")

	if answer, ok := memory.Repeat("say that again"); !ok || answer != "In the last seven days we took in 26400.00." {
		t.Fatalf("Repeat() = %q, %v", answer, ok)
	}
}

func TestRepeatEmptyMemory(t *testing.T) {
	memory := NewMemory(0)
	if _, ok := memory.Repeat("again"); ok {
		t.Fatal("Repeat() on empty memory should not match")
	}
}

func TestRespondBotIdentity(t *testing.T) {
	memory := NewMemory(0)
	for _, question := range []string{"what is your name?", "what are you?", "who are you?"} {
		answer, ok := memory.Respond(question)
		if !ok || !strings.Contains(answer, BotName) {
			t.Fatalf("Respond(%q) = %q, %v", question, answer, ok)
		}
	}
}

func TestRespondGreetingAndFarewell(t *testing.T) {
	memory := NewMemory(0)
	if answer, ok := memory.Respond("hello"); !ok || !strings.Contains(answer, BotName) {
		t.Fatalf("greeting = %q, %v", answer, ok)
	}
	if answer, ok := memory.Respond("goodbye"); !ok || answer == "" {
		t.Fatalf("farewell = %q, %v", answer, ok)
	}
}

func TestRespondLearnsAndRecallsName(t *testing.T) {
	memory := NewMemory(0)
	for _, tell := range []string{"my name is Ama", "I am Kojo", "call me Efua", "my name's Yaw"} {
		memory = NewMemory(0)
		if answer, ok := memory.Respond(tell); !ok || !strings.Contains(answer, "meet") {
			t.Fatalf("tell(%q) = %q, %v", tell, answer, ok)
		}
		answer, ok := memory.Respond("what is my name?")
		var expected string
		if strings.Contains(tell, "Ama") {
			expected = "Ama"
		} else if strings.Contains(tell, "Kojo") {
			expected = "Kojo"
		} else if strings.Contains(tell, "Efua") {
			expected = "Efua"
		} else {
			expected = "Yaw"
		}
		if !ok || !strings.Contains(answer, expected) {
			t.Fatalf("recall after %q = %q, %v", tell, answer, ok)
		}
	}
}

func TestRespondUnknownNameBeforeTelling(t *testing.T) {
	memory := NewMemory(0)
	answer, ok := memory.Respond("what is my name?")
	if !ok || !strings.Contains(answer, "don't know your name") {
		t.Fatalf("Respond() = %q, %v", answer, ok)
	}
}

func TestRespondIgnoresDataQuestions(t *testing.T) {
	memory := NewMemory(0)
	for _, question := range []string{
		"How much money did we receive in the last week?",
		"Which staff has been absent for a week?",
		"Show recent attendance records.",
	} {
		if _, ok := memory.Respond(question); ok {
			t.Fatalf("Respond(%q) should not match a data question", question)
		}
	}
}

func TestSmalltalkMood(t *testing.T) {
	memory := NewMemory(0)
	for _, question := range []string{"how are you?", "what about you?", "and you?"} {
		if answer, ok := memory.Respond(question); !ok || answer == "" {
			t.Fatalf("mood(%q) = %q, %v", question, answer, ok)
		}
	}
}

func TestSmalltalkFeelings(t *testing.T) {
	for _, question := range []string{
		"I'm doing well, what about you?",
		"I am happy today",
		"I'm a bit tired today",
		"I'm so stressed about work",
	} {
		memory := NewMemory(0)
		if answer, ok := memory.Respond(question); !ok || answer == "" {
			t.Fatalf("feeling(%q) = %q, %v", question, answer, ok)
		}
	}
}

func TestSmalltalkFoodAndPraise(t *testing.T) {
	memory := NewMemory(0)
	for _, question := range []string{
		"what would you like to eat?",
		"have you eaten?",
		"thank you",
		"you're the best",
	} {
		if answer, ok := memory.Respond(question); !ok || answer == "" {
			t.Fatalf("smalltalk(%q) = %q, %v", question, answer, ok)
		}
	}
}

func TestSmalltalkBotNature(t *testing.T) {
	memory := NewMemory(0)
	for _, question := range []string{"are you a robot?", "are you human?", "are you real?"} {
		if answer, ok := memory.Respond(question); !ok || answer == "" {
			t.Fatalf("nature(%q) = %q, %v", question, answer, ok)
		}
	}
}