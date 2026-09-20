package model

import (
	"path/filepath"
	"testing"
)

func TestTrainSaveLoadAndAnswer(t *testing.T) {
	knowledge := Knowledge{
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
}
