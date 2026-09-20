package embedding

import (
	"math"

	"nyansapo/tokenizer"
)

type Vector map[string]float64

func Embed(text string) Vector {
	vector := make(Vector)
	for _, token := range tokenizer.Tokenize(text) {
		if len(token) > 2 {
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
