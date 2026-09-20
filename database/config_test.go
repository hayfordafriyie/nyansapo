package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDotEnvDoesNotOverrideEnvironment(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("NYANSAPO_DB_PROVIDER=postgres\nNYANSAPO_DB_PASSWORD=secret\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NYANSAPO_DB_PROVIDER", "mysql")
	if err := LoadDotEnv(path); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("NYANSAPO_DB_PROVIDER"); got != "mysql" {
		t.Fatalf("provider = %q, want existing environment value", got)
	}
	if got := os.Getenv("NYANSAPO_DB_PASSWORD"); got != "secret" {
		t.Fatalf("password = %q, want dotenv value", got)
	}
}

func TestReadOnlyQueryValidation(t *testing.T) {
	for _, query := range []string{"SELECT * FROM users", "WITH totals AS (SELECT 1) SELECT * FROM totals", "SHOW TABLES"} {
		if err := ValidateReadOnlyQuery(query); err != nil {
			t.Fatalf("ValidateReadOnlyQuery(%q): %v", query, err)
		}
	}
	for _, query := range []string{"DELETE FROM users", "UPDATE users SET name='x'", "SELECT 1; DROP TABLE users"} {
		if err := ValidateReadOnlyQuery(query); err == nil {
			t.Fatalf("ValidateReadOnlyQuery(%q) accepted unsafe query", query)
		}
	}
}

func TestCatalogDocumentsIncludeRelationships(t *testing.T) {
	catalog := Catalog{
		Database: "shop",
		Tables: []Table{{
			Name:        "orders",
			Columns:     []Column{{Name: "id", Type: "integer", PrimaryKey: true}},
			ForeignKeys: []ForeignKey{{Column: "customer_id", ReferencedTable: "customers", ReferencedColumn: "id"}},
		}},
	}
	documents := catalog.Documents()
	if len(documents) != 1 || documents[0] == "" {
		t.Fatalf("Documents() = %#v, want catalog document", documents)
	}
	if !strings.Contains(documents[0], "references") {
		t.Fatalf("document = %q, want relationship text", documents[0])
	}
}

func TestSQLDriverAndDSN(t *testing.T) {
	mysql := Config{
		Provider: ProviderMySQL,
		Host:     "db.example",
		Port:     3306,
		User:     "reader",
		Password: "secret",
		Database: "app",
	}
	driver, dsn, err := mysql.SQLDriverAndDSN()
	if err != nil || driver != "mysql" || !strings.Contains(dsn, "db.example:3306") {
		t.Fatalf("mysql driver/dsn = %q, %q, %v", driver, dsn, err)
	}
	postgres := mysql
	postgres.Provider = ProviderPostgreSQL
	driver, dsn, err = postgres.SQLDriverAndDSN()
	if err != nil || driver != "pgx" || !strings.Contains(dsn, "sslmode=disable") {
		t.Fatalf("postgres driver/dsn = %q, %q, %v", driver, dsn, err)
	}
}
