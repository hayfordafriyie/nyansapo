package model

import (
	"encoding/json"
	"fmt"
	"os"

	"mini-llm/embedding"
)

type trainedCandidate struct {
	Text   string           `json:"text"`
	Vector embedding.Vector `json:"vector"`
}

type TrainedModel struct {
	Candidates []trainedCandidate `json:"candidates"`
}

func Train(knowledge Knowledge) TrainedModel {
	texts := append([]string{knowledge.Description}, knowledge.Features...)
	candidates := make([]trainedCandidate, 0, len(texts))
	for _, text := range texts {
		candidates = append(candidates, trainedCandidate{
			Text:   text,
			Vector: embedding.Embed(text),
		})
	}
	return TrainedModel{Candidates: candidates}
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
	return model, nil
}

func (m TrainedModel) Answer(question string) string {
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
	return best
}
