package pipeline

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var ErrNoDocuments = errors.New("no supported documents")

type Document struct {
	Source string
	Text   string
}

type Reader func(path string) ([]Document, error)

type Pipeline struct {
	readers map[string]Reader
}

func New() *Pipeline {
	p := &Pipeline{readers: make(map[string]Reader)}
	p.Register(".txt", readText)
	p.Register(".md", readText)
	p.Register(".markdown", readText)
	p.Register(".json", readJSON)
	p.Register(".csv", readCSV)
	return p
}

func (p *Pipeline) Register(extension string, reader Reader) {
	p.readers[strings.ToLower(extension)] = reader
}

func (p *Pipeline) Load(root string) ([]Document, error) {
	var documents []Document
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}

		reader, supported := p.readers[strings.ToLower(filepath.Ext(path))]
		if !supported {
			return nil
		}
		items, err := reader(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		documents = append(documents, items...)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("load documents: %w", err)
	}
	if len(documents) == 0 {
		return nil, fmt.Errorf("%w found in %s", ErrNoDocuments, root)
	}
	return documents, nil
}

func readText(path string) ([]Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return nil, nil
	}
	return []Document{{Source: path, Text: text}}, nil
}

func readJSON(path string) ([]Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	if catalogDocuments := readCatalogJSON(path, value); len(catalogDocuments) > 0 {
		return catalogDocuments, nil
	}
	if items, ok := value.([]any); ok {
		documents := make([]Document, 0, len(items))
		for _, item := range items {
			var texts []string
			collectStrings(item, &texts)
			content := strings.TrimSpace(strings.Join(texts, ". "))
			if content != "" {
				documents = append(documents, Document{Source: path, Text: content})
			}
		}
		return documents, nil
	}
	var texts []string
	collectStrings(value, &texts)
	var content []string
	for _, text := range texts {
		if text = strings.TrimSpace(text); text != "" {
			content = append(content, text)
		}
	}
	if len(content) == 0 {
		return nil, nil
	}
	return []Document{{Source: path, Text: strings.Join(content, ". ")}}, nil
}

func readCatalogJSON(path string, value any) []Document {
	root, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	var documents []Document
	for _, key := range []string{"tables", "Tables", "collections", "Collections"} {
		items, ok := root[key].([]any)
		if !ok {
			continue
		}
		for _, item := range items {
			content := catalogItemDocument(item, key == "collections" || key == "Collections")
			if content != "" {
				documents = append(documents, Document{Source: path, Text: content})
			}
		}
	}
	return documents
}

func catalogItemDocument(value any, collection bool) string {
	item, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	label := "table"
	if collection {
		label = "collection"
	}
	var parts []string
	if name := mapString(item, "name", "Name"); name != "" {
		parts = append(parts, label, name)
	}
	if schema := mapString(item, "schema", "Schema"); schema != "" {
		parts = append(parts, "schema", schema)
	}
	for _, key := range []string{"columns", "Columns", "fields", "Fields"} {
		if entries, ok := item[key].([]any); ok {
			parts = append(parts, "columns")
			for _, entry := range entries {
				if field, ok := entry.(map[string]any); ok {
					name := mapString(field, "name", "Name", "path", "Path")
					typ := mapString(field, "type", "Type")
					if name != "" {
						parts = append(parts, name, typ)
					}
				}
			}
		}
	}
	for _, key := range []string{"foreign_keys", "ForeignKeys"} {
		if entries, ok := item[key].([]any); ok {
			parts = append(parts, "relationships")
			var nested []string
			collectLabeledStrings(entries, &nested)
			parts = append(parts, nested...)
		}
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func mapString(item map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := item[key].(string); ok {
			return value
		}
	}
	return ""
}

func collectLabeledStrings(value any, texts *[]string) {
	switch typed := value.(type) {
	case string:
		*texts = append(*texts, typed)
	case []any:
		for _, item := range typed {
			collectLabeledStrings(item, texts)
		}
	case map[string]any:
		for key, item := range typed {
			switch nested := item.(type) {
			case string:
				*texts = append(*texts, key+" "+nested)
			default:
				collectLabeledStrings(nested, texts)
			}
		}
	}
}

func collectStrings(value any, texts *[]string) {
	switch typed := value.(type) {
	case string:
		*texts = append(*texts, typed)
	case []any:
		for _, item := range typed {
			collectStrings(item, texts)
		}
	case map[string]any:
		for key, item := range typed {
			if strings.ToLower(key) == "subject" {
				collectStrings(item, texts)
			}
		}
		for key, item := range typed {
			switch strings.ToLower(key) {
			case "subject", "name", "title", "topic":
				continue
			}
			collectStrings(item, texts)
		}
	}
}

func readCSV(path string) ([]Document, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		return nil, err
	}
	var header []string
	if len(rows) > 0 {
		header = make([]string, len(rows[0]))
		for index, value := range rows[0] {
			header[index] = strings.ToLower(strings.TrimSpace(value))
		}
	}
	documents := make([]Document, 0, len(rows))
	for rowIndex, row := range rows {
		if rowIndex == 0 && len(header) >= 2 && header[0] == "topic" && header[1] == "explanation" {
			continue
		}
		text := strings.TrimSpace(strings.Join(row, ": "))
		if len(header) >= 2 && header[0] == "topic" && header[1] == "explanation" && len(row) >= 2 {
			text = strings.TrimSpace(row[1])
		}
		if text != "" {
			documents = append(documents, Document{Source: path, Text: text})
		}
	}
	return documents, nil
}
