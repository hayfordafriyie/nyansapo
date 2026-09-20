package evaluation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"nyansapo/model"
)

var Questions = []string{
	"How are orders related to customers?",
	"Which columns identify a customer?",
	"Which columns store order totals?",
	"What is a primary key?",
	"What is a foreign key?",
	"How do I join customers and orders?",
	"What does GROUP BY do?",
	"What is a window function?",
	"What is a common table expression?",
	"How do I inspect a query plan?",
	"What is an index used for?",
	"What does information_schema contain?",
	"How does PostgreSQL support JSONB?",
	"How does MySQL expose metadata?",
	"How does MongoDB store nested fields?",
	"How does Cassandra use partition keys?",
	"Explain SQL joins",
	"Explain database normalization",
	"Explain read-only queries",
	"Explain aggregation",
}

type Result struct {
	Total      int
	Unknown    int
	Empty      int
	Ungrounded int
	Unique     int
	Samples    []string
}

func Run(trained model.TrainedModel, repetitions int) Result {
	answers := make(map[string]struct{})
	result := Result{}
	for run := 0; run < repetitions; run++ {
		for _, question := range Questions {
			answer := trained.AnswerResult(question)
			record(&result, answers, answer.Answer, answer.Grounded)
		}
	}
	result.Unique = len(answers)
	return result
}

func RunAPI(client *http.Client, endpoint string, repetitions int) (Result, error) {
	answers := make(map[string]struct{})
	result := Result{}
	for run := 0; run < repetitions; run++ {
		for _, question := range Questions {
			payload, err := json.Marshal(map[string]string{"question": question})
			if err != nil {
				return Result{}, err
			}
			request, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
			if err != nil {
				return Result{}, err
			}
			request.Header.Set("Content-Type", "application/json")
			response, err := client.Do(request)
			if err != nil {
				return Result{}, err
			}
			var body struct {
				Answer   string `json:"answer"`
				Grounded bool   `json:"grounded"`
			}
			decodeErr := json.NewDecoder(response.Body).Decode(&body)
			response.Body.Close()
			if response.StatusCode != http.StatusOK {
				return Result{}, fmt.Errorf("API returned status %d", response.StatusCode)
			}
			if decodeErr != nil {
				return Result{}, decodeErr
			}
			record(&result, answers, body.Answer, body.Grounded)
		}
	}
	result.Unique = len(answers)
	return result, nil
}

func record(result *Result, answers map[string]struct{}, answer string, grounded bool) {
	result.Total++
	if answer == "I do not know that yet." {
		result.Unknown++
	}
	if answer == "" {
		result.Empty++
	}
	if !grounded {
		result.Ungrounded++
	}
	answers[answer] = struct{}{}
	if len(result.Samples) < 5 {
		result.Samples = append(result.Samples, answer)
	}
}
