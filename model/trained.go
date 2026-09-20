package model

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"nyansapo/embedding"
	"nyansapo/response"
)

type trainedCandidate struct {
	Text   string           `json:"text"`
	Vector embedding.Vector `json:"vector"`
}

type TrainedModel struct {
	Candidates []trainedCandidate `json:"candidates"`
	Knowledge  Knowledge          `json:"knowledge"`
}

func TrainTexts(texts []string) TrainedModel {
	candidates := make([]trainedCandidate, 0, len(texts))
	for _, text := range texts {
		if strings.TrimSpace(text) == "" {
			continue
		}
		candidates = append(candidates, trainedCandidate{
			Text:   text,
			Vector: embedding.Embed(text),
		})
	}
	return TrainedModel{Candidates: candidates}
}

func Train(knowledge Knowledge) TrainedModel {
	model := TrainTexts(append([]string{knowledge.Description}, knowledge.Features...))
	model.Knowledge = knowledge
	return model
}

func (m TrainedModel) Save(path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("encode trained model: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write trained model: %w", err)
	}
	return nil
}

func LoadTrained(path string) (TrainedModel, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return TrainedModel{}, fmt.Errorf("read trained model: %w", err)
	}
	var model TrainedModel
	if err := json.Unmarshal(data, &model); err != nil {
		return TrainedModel{}, fmt.Errorf("parse trained model: %w", err)
	}
	if err := model.Validate(); err != nil {
		return TrainedModel{}, fmt.Errorf("validate trained model: %w", err)
	}
	return model, nil
}

func (m TrainedModel) Validate() error {
	if len(m.Candidates) == 0 {
		return fmt.Errorf("at least one candidate is required")
	}
	for index, candidate := range m.Candidates {
		if strings.TrimSpace(candidate.Text) == "" || len(candidate.Vector) == 0 {
			return fmt.Errorf("candidate %d is empty", index)
		}
	}
	return nil
}

func (m TrainedModel) Answer(question string) string {
	if answer := m.Knowledge.Answer(question); strings.TrimSpace(answer) != "" &&
		answer != "I do not know that yet." {
		return response.Format(question, answer)
	}

	query := embedding.Embed(question)
	var best string
	var bestScore float64
	for _, candidate := range m.Candidates {
		score := embedding.Cosine(query, candidate.Vector)
		if score > bestScore {
			bestScore = score
			best = candidate.Text
		}
	}
	if bestScore == 0 {
		return "I do not know that yet."
	}
	return response.Format(question, best)
}
