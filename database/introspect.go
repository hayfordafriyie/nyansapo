package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func IntrospectConfigured(ctx context.Context, config Config) (Catalog, error) {
	switch config.Provider {
	case ProviderMongoDB:
		return IntrospectMongo(ctx, config)
	case ProviderCassandra:
		return IntrospectCassandra(ctx, config)
	}
	driver, dsn, err := connectionDriverAndDSN(config)
	if err != nil {
		return Catalog{}, err
	}
	if !config.ReadOnly {
		return Catalog{}, fmt.Errorf("database introspection requires NYANSAPO_DB_READ_ONLY=true")
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return Catalog{}, fmt.Errorf("open database: %w", err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return Catalog{}, fmt.Errorf("connect to database: %w", err)
	}
	return IntrospectSQL(ctx, db, config.Provider, config.Database, config.Name)
}

func SaveCatalog(path string, catalog Catalog) error {
	data, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return fmt.Errorf("encode catalog: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write catalog: %w", err)
	}
	return nil
}
