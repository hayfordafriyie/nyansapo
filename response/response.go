package response

import (
	"hash/fnv"
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

	templates := []string{
		"In simple terms, %s",
		"Put simply, %s",
		"A helpful way to think about it is: %s",
		"Here is the key idea: %s",
	}
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(question))
	return strings.Replace(templates[hash.Sum32()%uint32(len(templates))], "%s", body, 1)
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

	best := scoredSentences[0]
	for _, candidate := range scoredSentences[1:] {
		if candidate.score > best.score {
			best = candidate
		}
	}
	if best.score == 0 || len(sentences) == 1 {
		return []string{best.text}
	}
	return []string{best.text}
}
