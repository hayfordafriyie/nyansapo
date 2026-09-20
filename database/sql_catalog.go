package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type Queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func IntrospectSQL(ctx context.Context, db Queryer, provider Provider, databaseName, schemaName string) (Catalog, error) {
	if provider != ProviderMySQL && provider != ProviderPostgreSQL {
		return Catalog{}, fmt.Errorf("SQL introspection does not support %s", provider)
	}
	tableQuery := `
		SELECT table_schema, table_name
		FROM information_schema.tables
		WHERE table_type = 'BASE TABLE'
		  AND (? = '' OR table_schema = ?)
		ORDER BY table_schema, table_name`
	if provider == ProviderPostgreSQL {
		tableQuery = `
		SELECT table_schema, table_name
		FROM information_schema.tables
		WHERE table_type = 'BASE TABLE'
		AND ($1 = '' OR table_schema = $1)
		ORDER BY table_schema, table_name`
	}
	var rows *sql.Rows
	var err error
	if provider == ProviderPostgreSQL {
		rows, err = db.QueryContext(ctx, tableQuery, schemaName)
	} else {
		rows, err = db.QueryContext(ctx, tableQuery, schemaName, schemaName)
	}
	if err != nil {
		return Catalog{}, fmt.Errorf("list tables: %w", err)
	}
	defer rows.Close()

	catalog := Catalog{Provider: provider, Database: databaseName}
	for rows.Next() {
		var table Table
		if err := rows.Scan(&table.Schema, &table.Name); err != nil {
			return Catalog{}, fmt.Errorf("scan table: %w", err)
		}
		if isSystemSchema(provider, table.Schema) {
			continue
		}
		catalog.Tables = append(catalog.Tables, table)
	}

	if err := rows.Err(); err != nil {
		return Catalog{}, fmt.Errorf("read tables: %w", err)
	}
	for index := range catalog.Tables {
		table := &catalog.Tables[index]
		if err := loadColumns(ctx, db, provider, table); err != nil {
			return Catalog{}, err
		}
		if err := loadForeignKeys(ctx, db, provider, table); err != nil {
			return Catalog{}, err
		}
	}

	return catalog, nil
}

func isSystemSchema(provider Provider, schema string) bool {
	schema = strings.ToLower(schema)
	if provider == ProviderPostgreSQL {
		return schema == "information_schema" || schema == "pg_catalog" || strings.HasPrefix(schema, "pg_toast")
	}
	return schema == "information_schema" || schema == "performance_schema" || schema == "mysql" || schema == "sys"
}

func loadForeignKeys(ctx context.Context, db Queryer, provider Provider, table *Table) error {
	query := `
		SELECT column_name, referenced_table_schema, referenced_table_name, referenced_column_name
		FROM information_schema.key_column_usage
		WHERE table_schema = ? AND table_name = ? AND referenced_table_name IS NOT NULL`
	if provider == ProviderPostgreSQL {
		query = `
		SELECT kcu.column_name, ccu.table_schema, ccu.table_name, ccu.column_name
		FROM information_schema.key_column_usage kcu
		JOIN information_schema.constraint_column_usage ccu
		  ON kcu.constraint_name = ccu.constraint_name
		 AND kcu.constraint_schema = ccu.constraint_schema
		WHERE kcu.table_schema = $1 AND kcu.table_name = $2`
	}
	rows, err := db.QueryContext(ctx, query, table.Schema, table.Name)
	if err != nil {
		return fmt.Errorf("list foreign keys for %s.%s: %w", table.Schema, table.Name, err)
	}
	defer rows.Close()
	for rows.Next() {
		var key ForeignKey
		if err := rows.Scan(&key.Column, &key.ReferencedSchema, &key.ReferencedTable, &key.ReferencedColumn); err != nil {
			return fmt.Errorf("scan foreign key for %s.%s: %w", table.Schema, table.Name, err)
		}
		table.ForeignKeys = append(table.ForeignKeys, key)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read foreign keys for %s.%s: %w", table.Schema, table.Name, err)
	}
	return nil
}

func loadColumns(ctx context.Context, db Queryer, provider Provider, table *Table) error {
	query := `
		SELECT column_name, data_type, is_nullable, COALESCE(column_default, '')
		FROM information_schema.columns
		WHERE table_schema = ? AND table_name = ?
		ORDER BY ordinal_position`
	if table.Schema == "" {
		return fmt.Errorf("table schema is required for %s", table.Name)
	}
	if provider == ProviderPostgreSQL {
		query = `
		SELECT column_name, data_type, is_nullable, COALESCE(column_default, '')
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position`
	}
	rows, err := db.QueryContext(ctx, query, table.Schema, table.Name)
	if err != nil {
		return fmt.Errorf("list columns for %s.%s: %w", table.Schema, table.Name, err)
	}
	defer rows.Close()
	for rows.Next() {
		var column Column
		var nullable string
		if err := rows.Scan(&column.Name, &column.Type, &nullable, &column.Default); err != nil {
			return fmt.Errorf("scan column for %s.%s: %w", table.Schema, table.Name, err)
		}
		column.Nullable = nullable == "YES"
		table.Columns = append(table.Columns, column)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read columns for %s.%s: %w", table.Schema, table.Name, err)
	}
	return nil
}
