package pipeline

import (
	"os"
	"path/filepath"
	"strings"
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

func TestLoadJSONArrayCreatesSearchableDocuments(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "catalog.json")
	content := `["customers has an id column", "orders has a total column"]`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	documents, err := New().Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(documents) != 2 {
		t.Fatalf("got %d documents, want one per catalog entry", len(documents))
	}
}

func TestLoadCatalogJSONCreatesOneDocumentPerTable(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "catalog.json")
	content := `{"tables":[{"name":"customers","columns":[{"name":"id","type":"bigint"}]},{"name":"orders","columns":[{"name":"customer_id","type":"bigint"}]}]}`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	documents, err := New().Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(documents) != 2 || !strings.Contains(documents[0].Text, "customers") {
		t.Fatalf("documents = %+v, want one labeled document per table", documents)
	}
}
