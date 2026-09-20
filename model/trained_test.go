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
