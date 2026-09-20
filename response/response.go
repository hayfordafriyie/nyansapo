package response

import (
	"strings"

	"mini-llm/tokenizer"
)

func Format(question, passage string) string {
	sentences := splitSentences(passage)
	if len(sentences) == 0 {
		return passage
	}

	relevant := bestSentences(question, sentences)
	body := strings.Join(relevant, " ")
	if len(relevant) == 1 && len(strings.Fields(body)) <= 8 {
		return strings.TrimRight(body, ".!?")
	}

	switch {
	case strings.Contains(strings.ToLower(question), "why"):
		return "The reason is that " + body
	case strings.Contains(strings.ToLower(question), "how"):
		return "The process works like this: " + body
	case strings.Contains(strings.ToLower(question), "what"):
		return "In simple terms, " + body
	default:
		return "The key idea is: " + body
	}
}

func splitSentences(text string) []string {
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return r == '.' || r == '!' || r == '?' || r == '\n'
	})
	sentences := make([]string, 0, len(fields))
	for _, sentence := range fields {
		if sentence = strings.TrimSpace(sentence); sentence != "" {
			sentences = append(sentences, sentence+".")
		}
	}
	return sentences
}

func bestSentences(question string, sentences []string) []string {
	query := tokenizer.Tokenize(question)
	type scored struct {
		text  string
		score int
		index int
	}
	scoredSentences := make([]scored, 0, len(sentences))
	for index, sentence := range sentences {
		score := 0
		tokens := tokenizer.Tokenize(sentence)
		for _, queryToken := range query {
			for _, token := range tokens {
				if queryToken == token && len(token) > 2 {
					score++
					break
				}
			}
		}
		scoredSentences = append(scoredSentences, scored{text: sentence, score: score, index: index})
	}

	bestIndex := 0
	for _, candidate := range scoredSentences[1:] {
		if candidate.score > scoredSentences[bestIndex].score {
			bestIndex = candidate.index
		}
	}
	best := scoredSentences[bestIndex]
	if best.score == 0 || len(sentences) == 1 {
		return []string{best.text}
	}

	relevant := []string{best.text}
	if best.index+1 < len(sentences) {
		relevant = append(relevant, sentences[best.index+1])
	}
	return relevant
}
