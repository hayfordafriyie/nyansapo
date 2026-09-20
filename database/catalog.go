package database

import (
	"fmt"
	"strings"
)

type Catalog struct {
	Provider    Provider
	Name        string
	Database    string
	Tables      []Table
	Collections []Collection
}

type Table struct {
	Schema      string
	Name        string
	Columns     []Column
	PrimaryKey  []string
	ForeignKeys []ForeignKey
}

type Column struct {
	Name       string
	Type       string
	Nullable   bool
	Default    string
	PrimaryKey bool
}

type ForeignKey struct {
	Column           string
	ReferencedSchema string
	ReferencedTable  string
	ReferencedColumn string
}

type Collection struct {
	Name   string
	Fields []Field
}

type Field struct {
	Path string
	Type string
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
