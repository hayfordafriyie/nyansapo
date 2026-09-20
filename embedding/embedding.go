package embedding

import (
	"math"

	"nyansapo/tokenizer"
)

type Vector map[string]float64

var stopWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "are": {}, "as": {}, "at": {}, "be": {},
	"by": {}, "do": {}, "does": {}, "for": {}, "from": {}, "how": {}, "i": {},
	"in": {}, "is": {}, "it": {}, "of": {}, "on": {}, "or": {}, "that": {},
	"the": {}, "this": {}, "to": {}, "was": {}, "what": {}, "when": {},
	"where": {}, "which": {}, "who": {}, "why": {}, "with": {}, "you": {},
}

func Embed(text string) Vector {
	vector := make(Vector)
	for _, token := range tokenizer.Tokenize(text) {
		if len(token) > 2 {
			if _, excluded := stopWords[token]; excluded {
				continue
			}
			vector[token]++
		}
	}
	return vector
}

func Cosine(left, right Vector) float64 {
	var dot, leftMagnitude, rightMagnitude float64
	for token, leftValue := range left {
		rightValue := right[token]
		dot += leftValue * rightValue
		leftMagnitude += leftValue * leftValue
	}
	for _, rightValue := range right {
		rightMagnitude += rightValue * rightValue
	}
	if leftMagnitude == 0 || rightMagnitude == 0 {
		return 0
	}
	return dot / (math.Sqrt(leftMagnitude) * math.Sqrt(rightMagnitude))
}
