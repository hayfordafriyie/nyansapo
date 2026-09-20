package model

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"mini-llm/tokenizer"
)

type Knowledge struct {
	Name        string   `json:"name"`
	Country     string   `json:"country"`
	Description string   `json:"description"`
	Features    []string `json:"features"`
}

func LoadKnowledge(path string) (Knowledge, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Knowledge{}, fmt.Errorf("read knowledge: %w", err)
	}

	var knowledge Knowledge
	if err := json.Unmarshal(data, &knowledge); err != nil {
		return Knowledge{}, fmt.Errorf("parse knowledge: %w", err)
	}

	return knowledge, nil
}

func (k Knowledge) Answer(question string) string {
	tokens := tokenizer.Tokenize(question)
	normalized := strings.Join(tokens, " ")
	switch {
	case strings.Contains(normalized, "what is") && strings.Contains(normalized, "edspike"):
		return k.Description
	case strings.Contains(normalized, "country"):
		return k.Country
	case strings.Contains(normalized, "feature"):
		return strings.Join(k.Features, ", ")
	case strings.Contains(normalized, "name"):
		return k.Name
	default:
		return k.Search(tokens)
	}
}

func (k Knowledge) Search(query []string) string {
	for _, feature := range k.Features {
		if hasSharedToken(query, tokenizer.Tokenize(feature)) {
			return feature
		}
	}

	if hasSharedToken(query, tokenizer.Tokenize(k.Description)) {
		return k.Description
	}

	return "I do not know that yet."
}

func hasSharedToken(left, right []string) bool {
	for _, leftToken := range left {
		for _, rightToken := range right {
			if leftToken == rightToken && len(leftToken) > 2 {
				return true
			}
		}
	}
	return false
}
