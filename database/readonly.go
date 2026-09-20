package database

import (
	"fmt"
	"strings"
	"unicode"
)

func ValidateReadOnlyQuery(query string) error {
	tokens := queryTokens(query)
	if len(tokens) == 0 {
		return fmt.Errorf("query is empty")
	}
	if strings.Contains(query, ";") {
		return fmt.Errorf("multiple statements are not allowed")
	}
	switch tokens[0] {
	case "select", "with", "show", "describe", "desc", "explain":
		return nil
	default:
		return fmt.Errorf("only read-only queries are allowed")
	}
}

func queryTokens(query string) []string {
	var tokens []string
	var builder strings.Builder
	for _, character := range strings.ToLower(query) {
		if unicode.IsLetter(character) {
			builder.WriteRune(character)
			continue
		}
		if builder.Len() > 0 {
			tokens = append(tokens, builder.String())
			builder.Reset()
		}
	}
	if builder.Len() > 0 {
		tokens = append(tokens, builder.String())
	}
	return tokens
}
