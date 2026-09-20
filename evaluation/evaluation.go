package evaluation

import (
	"mini-llm/model"
)

var Questions = []string{
	"What is photosynthesis?",
	"How do plants make food?",
	"Why is chlorophyll important?",
	"What is a hypothesis?",
	"What is a fraction?",
	"What is the area of a rectangle?",
	"What is the mean?",
	"What is a prime number?",
	"What is a community?",
	"What does government do?",
	"What is geography?",
	"What is good citizenship?",
	"What is a noun?",
	"What is a verb?",
	"What is a paragraph?",
	"Explain science",
	"Explain mathematics",
	"Explain social studies",
	"Explain English",
	"How does evidence help science?",
}

type Result struct {
	Total   int
	Unknown int
	Empty   int
	Unique  int
	Samples []string
}

func Run(trained model.TrainedModel, repetitions int) Result {
	answers := make(map[string]struct{})
	result := Result{}
	for run := 0; run < repetitions; run++ {
		for _, question := range Questions {
			answer := trained.Answer(question)
			result.Total++
			if answer == "I do not know that yet." {
				result.Unknown++
			}
			if answer == "" {
				result.Empty++
			}
			answers[answer] = struct{}{}
			if len(result.Samples) < 5 {
				result.Samples = append(result.Samples, answer)
			}
		}
	}
	result.Unique = len(answers)
	return result
}
