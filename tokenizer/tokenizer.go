package tokenizer

import "strings"

func Tokenize(text string) []string {
	return strings.Fields(strings.ToLower(text))
}
