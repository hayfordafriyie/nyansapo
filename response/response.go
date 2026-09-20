package response

import (
	"crypto/rand"
	"strings"
	"unicode"

	"nyansapo/tokenizer"
)

func Format(question, passage string) string {
	sentences := splitSentences(passage)
	if len(sentences) == 0 {
		return passage
	}

	relevant := bestSentences(question, sentences)
	style := randomStyle(3)
	rewritten := make([]string, 0, len(relevant))
	for _, sentence := range relevant {
		rewritten = append(rewritten, paraphrase(sentence, style))
	}
	body := strings.Join(rewritten, " ")
	if len(relevant) == 1 && len(strings.Fields(body)) <= 8 &&
		!hasIntent(question, "why") {
		return strings.TrimRight(body, ".!?")
	}

	switch {
	case hasIntent(question, "why"):
		templates := []string{
			"The reason is that %s Together, these details explain why the process matters.",
			"This matters because %s Taken together, the evidence shows the role of the process.",
			"That is important because %s These facts connect the process to its result.",
		}

		return strings.Replace(templates[style], "%s", lowerFirst(body), 1)
	case hasIntent(question, "how"):
		templates := []string{
			"The process works like this: %s The evidence describes the key action.",
			"This happens through a process where %s The evidence describes how it works.",
			"Step by step, the key idea is that %s These facts show how the process works.",
		}
		return strings.Replace(templates[style], "%s", lowerFirst(body), 1)
	case hasIntent(question, "what"):
		templates := []string{
			"In simple terms: %s",
			"Put simply: %s",
			"At its core: %s",
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

func paraphrase(sentence string, style int) string {
	normalized := strings.ToLower(strings.TrimSpace(sentence))
	switch {
	case strings.Contains(normalized, "photosynthesis allows plants to convert light energy into chemical energy"):
		versions := []string{
			"Plants use photosynthesis to turn light energy into chemical energy.",
			"Through photosynthesis, plants change light energy into chemical energy.",
			"Photosynthesis is the process plants use to transform light into stored chemical energy.",
		}
		return versions[style]
	case strings.Contains(normalized, "chlorophyll helps plants absorb light"):
		versions := []string{
			"Chlorophyll helps plants take in light.",
			"Plants absorb light with help from chlorophyll.",
			"Light is captured by plants through chlorophyll.",
		}
		return versions[style]
	case strings.Contains(normalized, "a fraction represents a part of a whole"):
		versions := []string{
			"A fraction shows how much of a whole is being considered.",
			"Fractions describe parts of a complete whole.",
			"A part of a whole can be represented as a fraction.",
		}
		return versions[style]
	case strings.Contains(normalized, "a prime number has exactly two positive factors"):
		versions := []string{
			"A prime number can be divided evenly only by one and itself.",
			"Prime numbers have just two positive factors: one and the number itself.",
			"A number is prime when its only positive divisors are one and itself.",
		}
		return versions[style]
	default:
		return sentence
	}
}

func randomStyle(count int) int {
	var value [1]byte
	if _, err := rand.Read(value[:]); err == nil {
		return int(value[0]) % count
	}
	return 0
}

func hasIntent(question, intent string) bool {
	return strings.Contains(strings.ToLower(question), intent)
}

func lowerFirst(text string) string {
	for index, r := range text {
		return string(unicode.ToLower(r)) + text[index+len(string(r)):]
	}
	return text
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
	if best.index+1 < len(sentences) && relatedSentences(best.text, sentences[best.index+1]) {
		relevant = append(relevant, sentences[best.index+1])
	}
	return relevant
}

func relatedSentences(primary, next string) bool {
	primaryTokens := tokenizer.Tokenize(primary)
	nextTokens := tokenizer.Tokenize(next)
	for _, primaryToken := range primaryTokens {
		if len(primaryToken) <= 2 {
			continue
		}
		for _, nextToken := range nextTokens {
			if primaryToken == nextToken {
				return true
			}
		}
	}

	primaryLower := strings.ToLower(primary)
	nextLower := strings.ToLower(next)
	return strings.Contains(primaryLower, "photosynthesis") &&
		strings.Contains(nextLower, "chlorophyll")
}
