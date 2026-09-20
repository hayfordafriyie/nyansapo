package tokenizer

import (
	"strings"
	"unicode"
)

func Tokenize(text string) []string {
	var builder strings.Builder
	for _, character := range strings.ToLower(text) {
		if unicode.IsLetter(character) || unicode.IsNumber(character) {
			builder.WriteRune(character)
			continue
		}
		builder.WriteRune(' ')
	}

	return strings.Fields(builder.String())
}
