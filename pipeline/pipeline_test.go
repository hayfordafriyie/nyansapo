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
	if len(documents) != 4 {
		t.Fatalf("got %d documents, want 4", len(documents))
	}
	if documents[3].Text != "Ama: teacher" {
		t.Fatalf("CSV row = %q, want labeled row", documents[3].Text)
	}
}

func TestLoadTopicExplanationCSVUsesNaturalExplanation(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "subjects.csv")
	content := "topic,explanation\ncitizenship,\"Good citizenship includes participation, responsibility, and respect.\"\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	documents, err := New().Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(documents) != 1 || documents[0].Text != "Good citizenship includes participation, responsibility, and respect." {
		t.Fatalf("documents = %+v, want natural explanation", documents)
	}
}

func TestLoadJSONSkipsMetadataLabels(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "math.json")
	content := `{"subject":"Mathematics","topics":["A fraction represents a part of a whole."]}`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	documents, err := New().Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(documents) != 1 || documents[0].Text != "Mathematics. A fraction represents a part of a whole." {
		t.Fatalf("documents = %+v, want subject context with topic", documents)
	}
}
