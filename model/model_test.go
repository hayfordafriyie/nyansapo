package model

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadKnowledgeAndAnswer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "knowledge.json")
	content := `{"name":"EDSPiKE","country":"Ghana","description":"A platform.","features":["Learning","School management"]}`
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
	if got := knowledge.Answer("What does EDSPiKE do?"); got != "A platform." {
		t.Fatalf("Answer() = %q, want %q", got, "A platform.")
	}

	if got := knowledge.Answer("Tell me about learning"); got != "Learning" {
		t.Fatalf("Answer() = %q, want %q", got, "Learning")
	}

	if got := knowledge.Answer("Tell me about school management"); got != "School management" {
		t.Fatalf("Answer() = %q, want %q", got, "School management")
	}
}
