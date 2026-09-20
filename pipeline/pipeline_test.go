package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMixedDocuments(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("school operations"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "records.json"), []byte(`{"topic":"learning","items":["teachers"]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "rows.csv"), []byte("name,role\nAma,teacher\n"), 0600); err != nil {
		t.Fatal(err)
	}

	documents, err := New().Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(documents) != 7 {
		t.Fatalf("got %d documents, want 7", len(documents))
	}
}
