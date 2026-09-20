package model

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"nyansapo/embedding"
	"nyansapo/tokenizer"
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
	case strings.Contains(normalized, "edspike") &&
		(strings.Contains(normalized, "what is") ||
			strings.Contains(normalized, "what does") ||
			strings.Contains(normalized, "describe") ||
			strings.Contains(normalized, "platform")):
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
	type candidate struct {
		text  string
		score float64
	}

	queryVector := embedding.Embed(strings.Join(query, " "))
	candidates := make([]candidate, 0, len(k.Features)+1)
	for _, feature := range k.Features {
		candidates = append(candidates, candidate{
			text:  feature,
			score: embedding.Cosine(queryVector, embedding.Embed(feature)),
		})
	}
	candidates = append(candidates, candidate{
		text:  k.Description,
		score: embedding.Cosine(queryVector, embedding.Embed(k.Description)),
	})

	best := candidate{}
	for _, current := range candidates {
		if current.score > best.score {
			best = current
		}
	}
	if best.score > 0 {
		return best.text
	}

	return "I do not know that yet."
}
