package evaluation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"nyansapo/model"
)

var Questions = []string{
	"Which staff has been absent for a week?",
	"How much money did we receive in the last week?",
	"How many staff are recorded?",
	"How many payments were received?",
	"Show staff attendance from the last week.",
	"Show recent absences.",
	"Which employees were marked absent?",
	"What was our total revenue this week?",
	"What was our total revenue in the last seven days?",
	"How much was paid in the last week?",
	"How many sales do we have?",
	"How many orders were placed?",
	"How many invoices are recorded?",
	"How many transactions are recorded?",
	"Show recent payment totals.",
	"Which staff members have attendance records?",
	"Show business activity from the last week.",
	"How many customers do we have?",
	"How many active employees do we have?",
	"What data is available for staff attendance?",
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

func RunWith(answer func(string) string, repetitions int) Result {
	answers := make(map[string]struct{})
	result := Result{}
	for run := 0; run < repetitions; run++ {
		for _, question := range Questions {
			record(&result, answers, answer(question), true)
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
