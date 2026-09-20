package model

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadKnowledgeAndAnswer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "knowledge.json")
	content := `{"name":"EDSPiKE","country":"Ghana","description":"A platform.","features":["Learning"]}`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	knowledge, err := LoadKnowledge(path)
	if err != nil {
		t.Fatal(err)
	}

	if got := knowledge.Answer("What is EDSPiKE?"); got != "A platform." {
		t.Fatalf("Answer() = %q, want %q", got, "A platform.")
	}
}
