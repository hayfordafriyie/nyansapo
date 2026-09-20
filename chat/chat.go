package chat

import (
	"strings"

	"mini-llm/tokenizer"
)

type Knowledge struct {
	Name        string
	Country     string
	Description string
	Features    []string
}

func Answer(question string, knowledge Knowledge) string {
	tokens := tokenizer.Tokenize(question)
	normalized := strings.Join(tokens, " ")

	switch {
	case strings.Contains(normalized, "what is") && strings.Contains(normalized, "edspike"):
		return knowledge.Description
	case strings.Contains(normalized, "country"):
		return knowledge.Country
	case strings.Contains(normalized, "feature"):
		return strings.Join(knowledge.Features, ", ")
	default:
		return "I do not know that yet."
	}
}
