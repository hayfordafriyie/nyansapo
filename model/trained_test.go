package model

import (
	"path/filepath"
	"testing"
)

func TestTrainSaveLoadAndAnswer(t *testing.T) {
	knowledge := Knowledge{
		Name:        "EDSPiKE",
		Country:     "Ghana",
		Description: "A platform.",
		Features:    []string{"School management", "Learning management"},
	}

	trained := Train(knowledge)
	path := filepath.Join(t.TempDir(), "model.json")
	if err := trained.Save(path); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadTrained(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.Answer("school management"); got != "School management" {
		t.Fatalf("Answer() = %q, want %q", got, "School management")
	}
	if got := loaded.Answer("What country?"); got != "Ghana" {
		t.Fatalf("Answer() = %q, want %q", got, "Ghana")
	}
}

func TestValidateRejectsEmptyModel(t *testing.T) {
	if err := (TrainedModel{}).Validate(); err == nil {
		t.Fatal("Validate() accepted an empty model")
	}
}

func TestTrainTexts(t *testing.T) {
	trained := TrainTexts([]string{"school operations", "", "teacher management"})
	if len(trained.Candidates) != 2 {
		t.Fatalf("got %d candidates, want 2", len(trained.Candidates))
	}
}

func TestGenericModelDoesNotReturnEmptyMetadata(t *testing.T) {
	trained := TrainTexts([]string{"EDSPiKE is a school management platform in Ghana."})
	if got := trained.Answer("What is EDSPiKE?"); got == "" {
		t.Fatal("generic model returned an empty answer")
	}
}

func TestAnswerResultRejectsWeakRetrieval(t *testing.T) {
	trained := TrainTexts([]string{"Photosynthesis converts light energy into chemical energy."})
	result := trained.AnswerResult("What is quantum physics?")
	if result.Grounded {
		t.Fatalf("weak retrieval was marked grounded: %+v", result)
	}
	if result.Answer != "I do not know that yet." {
		t.Fatalf("weak retrieval answer = %q, want refusal", result.Answer)
	}
}

func TestAnswerResultRejectsSingleSharedTokenEcho(t *testing.T) {
	trained := TrainTexts([]string{"Your self-control in this respect will be the only witness."})
	result := trained.AnswerResult("What is your name?")
	if result.Grounded {
		t.Fatalf("single-token echo was marked grounded: %+v", result)
	}
	if result.Answer != "I do not know that yet." {
		t.Fatalf("single-token echo answer = %q, want refusal", result.Answer)
	}
}

func TestAnswerResultAcceptsMultiTokenMatch(t *testing.T) {
	trained := TrainTexts([]string{"The primary key uniquely identifies each row in a table."})
	result := trained.AnswerResult("What is a primary key?")
	if !result.Grounded {
		t.Fatalf("multi-token match was rejected: %+v", result)
	}
}

func TestAnswerResultIncludesEvidence(t *testing.T) {
	trained := TrainTexts([]string{"Photosynthesis converts light energy into chemical energy."})
	result := trained.AnswerResult("What is photosynthesis?")
	if !result.Grounded {
		t.Fatalf("supported retrieval was not grounded: %+v", result)
	}
	if result.Evidence == "" || result.Confidence <= 0 {
		t.Fatalf("result missing evidence or confidence: %+v", result)
	}
}

func TestTeachStoresAndAnswersFact(t *testing.T) {
	trained := TrainTexts([]string{"unrelated text about plants"})
	trained.Teach("1+1", "2")
	if answer := trained.Answer("1+1"); answer != "2" {
		t.Fatalf("taught fact answer = %q, want 2", answer)
	}
}

func TestTeachPersistsThroughSaveLoad(t *testing.T) {
	trained := TrainTexts([]string{"a candidate document"})
	trained.Teach("What is the capital of Ghana?", "Accra")
	path := filepath.Join(t.TempDir(), "model.json")
	if err := trained.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadTrained(path)
	if err != nil {
		t.Fatal(err)
	}
	if answer := loaded.Answer("what is the capital of ghana?"); answer != "Accra" {
		t.Fatalf("loaded taught fact = %q, want Accra", answer)
	}
}

func TestFactMatchesRephrasedQuestion(t *testing.T) {
	trained := TrainTexts([]string{"candidate text"})
	trained.Teach("1+1", "2")
	for _, question := range []string{"1+1", "What is 1+1?", "please compute 1 + 1"} {
		if answer := trained.Answer(question); answer != "2" {
			t.Fatalf("rephrased %q = %q, want 2", question, answer)
		}
	}
}

func TestFactDoesNotMatchDifferentQuestion(t *testing.T) {
	trained := TrainTexts([]string{"candidate text"})
	trained.Teach("1+1", "2")
	if answer := trained.Answer("2+2"); answer == "2" {
		t.Fatal("unrelated question matched the taught fact")
	}
}

func TestFactMatchingRequiresCoverage(t *testing.T) {
	trained := TrainTexts([]string{"candidate text"})
	trained.Teach("what is React?", "React is a JavaScript library.")
	if answer := trained.Answer("what is a component in React?"); answer != "I do not know that yet." {
		t.Fatalf("low-coverage question matched the React fact: %q", answer)
	}
	if answer := trained.Answer("what is React?"); answer != "React is a JavaScript library." {
		t.Fatalf("exact React question did not match: %q", answer)
	}
}
