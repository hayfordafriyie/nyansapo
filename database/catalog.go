package database

import (
	"fmt"
	"strings"
)

type Catalog struct {
	Provider    Provider     `json:"provider"`
	Name        string       `json:"name"`
	Database    string       `json:"database"`
	Tables      []Table      `json:"tables"`
	Collections []Collection `json:"collections"`
}

type Table struct {
	Schema      string       `json:"schema"`
	Name        string       `json:"name"`
	Columns     []Column     `json:"columns"`
	PrimaryKey  []string     `json:"primary_key"`
	ForeignKeys []ForeignKey `json:"foreign_keys"`
}

type Column struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Nullable   bool   `json:"nullable"`
	Default    string `json:"default"`
	PrimaryKey bool   `json:"primary_key"`
}

type ForeignKey struct {
	Column           string `json:"column"`
	ReferencedSchema string `json:"referenced_schema"`
	ReferencedTable  string `json:"referenced_table"`
	ReferencedColumn string `json:"referenced_column"`
}

type Collection struct {
	Name   string  `json:"name"`
	Fields []Field `json:"fields"`
}

type Field struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

func (c Catalog) Documents() []string {
	documents := make([]string, 0, len(c.Tables)+len(c.Collections))
	for _, table := range c.Tables {
		var builder strings.Builder
		fmt.Fprintf(&builder, "database %s table %s", c.Database, table.Name)
		if table.Schema != "" {
			fmt.Fprintf(&builder, " schema %s", table.Schema)
		}
		if len(table.Columns) > 0 {
			builder.WriteString(". columns:")
			for _, column := range table.Columns {
				fmt.Fprintf(&builder, " %s %s", column.Name, column.Type)
				if column.PrimaryKey {
					builder.WriteString(" primary_key")
				}
				if !column.Nullable {
					builder.WriteString(" not_null")
				}
			}
		}
		for _, key := range table.ForeignKeys {
			fmt.Fprintf(&builder, ". relationship %s references %s.%s.%s",
				key.Column, key.ReferencedSchema, key.ReferencedTable, key.ReferencedColumn)
		}
		documents = append(documents, builder.String())
	}
	for _, collection := range c.Collections {
		var builder strings.Builder
		fmt.Fprintf(&builder, "database %s collection %s", c.Database, collection.Name)
		for _, field := range collection.Fields {
			fmt.Fprintf(&builder, " field %s %s", field.Path, field.Type)
		}
		documents = append(documents, builder.String())
	}
	return documents
}
