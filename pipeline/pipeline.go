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

func collectStrings(value any, texts *[]string) {
	switch typed := value.(type) {
	case string:
		*texts = append(*texts, typed)
	case []any:
		for _, item := range typed {
			collectStrings(item, texts)
		}
	case map[string]any:
		for _, item := range typed {
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
	documents := make([]Document, 0, len(rows))
	for _, row := range rows {
		text := strings.TrimSpace(strings.Join(row, " "))
		if text != "" {
			documents = append(documents, Document{Source: path, Text: text})
		}
	}
	return documents, nil
}
