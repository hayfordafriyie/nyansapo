package response

import (
	"crypto/rand"
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

	style := randomStyle(3)
	switch {
	case strings.Contains(strings.ToLower(question), "why"):
		templates := []string{
			"The reason is that %s",
			"This matters because %s",
			"That is important because %s",
		}
		return strings.Replace(templates[style], "%s", body, 1)
	case strings.Contains(strings.ToLower(question), "how"):
		templates := []string{
			"The process works like this: %s",
			"Plants do this through a process where %s",
			"Step by step, the key idea is that %s",
		}
		return strings.Replace(templates[style], "%s", body, 1)
	case strings.Contains(strings.ToLower(question), "what"):
		templates := []string{
			"In simple terms, %s",
			"Put simply, %s",
			"At its core, %s",
		}
		return strings.Replace(templates[style], "%s", body, 1)
	default:
		templates := []string{
			"The key idea is: %s",
			"One useful way to see it is: %s",
			"In short, %s",
		}
		return strings.Replace(templates[style], "%s", body, 1)
	}
}

func randomStyle(count int) int {
	var value [1]byte
	if _, err := rand.Read(value[:]); err == nil {
		return int(value[0]) % count
	}
	return 0
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
