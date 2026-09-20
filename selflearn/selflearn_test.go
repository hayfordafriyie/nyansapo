package selflearn

import (
	"os"
	"path/filepath"
	"testing"

	"nyansapo/database"
	"nyansapo/model"
)

func TestTrainerTeachesAndPreservesFactsAcrossRefresh(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.json")
	source := filepath.Join(dir, "input")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	doc := `The company sells widgets stored in a products table.`
	if err := os.WriteFile(filepath.Join(source, "ops.md"), []byte(doc), 0o600); err != nil {
		t.Fatal(err)
	}

	initial := model.TrainTexts([]string{doc})
	trainer := New(Options{Source: source, ModelPath: modelPath, Interval: 3600}, initial)
	if err := trainer.Teach("what is 2+2", "4"); err != nil {
		t.Fatal(err)
	}
	if got := trainer.Answer("what is 2+2"); got != "4" {
		t.Fatalf("after teach, answer = %q, want 4", got)
	}

	if err := trainer.Refresh(); err != nil {
		t.Fatal(err)
	}
	if got := trainer.Answer("what is 2+2"); got != "4" {
		t.Fatalf("after refresh, fact lost: answer = %q, want 4", got)
	}
	if _, err := os.Stat(modelPath); err != nil {
		t.Fatalf("retrained model not persisted: %v", err)
	}
}

func TestTrainerRefreshSkipsUnchangedCatalog(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "input")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	doc := `The company sells widgets stored in a products table.`
	if err := os.WriteFile(filepath.Join(source, "ops.md"), []byte(doc), 0o600); err != nil {
		t.Fatal(err)
	}

	trainer := New(Options{Source: source, ModelPath: filepath.Join(dir, "model.json")}, model.TrainTexts([]string{doc}))
	first, err := os.Stat(filepath.Join(dir, "model.json"))
	if os.IsNotExist(err) {
		if err := trainer.Refresh(); err != nil {
			t.Fatal(err)
		}
		first, err = os.Stat(filepath.Join(dir, "model.json"))
		if err != nil {
			t.Fatal(err)
		}
	}

	if err := trainer.Refresh(); err != nil {
		t.Fatal(err)
	}
	second, err := os.Stat(filepath.Join(dir, "model.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !first.ModTime().Equal(second.ModTime()) {
		t.Fatalf("unchanged catalog triggered a rewrite: %v vs %v", first.ModTime(), second.ModTime())
	}
}

func TestCatalogSignatureDetectsNewTables(t *testing.T) {
	base := database.Catalog{Provider: database.ProviderPostgreSQL, Database: "d"}
	base.Tables = []database.Table{{Schema: "public", Name: "employees", Columns: []database.Column{{Name: "id", Type: "integer"}}}}
	same := base
	if got, want := catalogSignature(base), catalogSignature(same); got != want {
		t.Fatalf("identical catalogs have different signatures: %q vs %q", got, want)
	}
	changed := base
	changed.Tables = append(changed.Tables, database.Table{Schema: "public", Name: "product_feedback"})
	if got, want := catalogSignature(changed), catalogSignature(base); got == want {
		t.Fatalf("new table not reflected in signature")
	}
}