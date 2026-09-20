package model

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"nyansapo/embedding"
	"nyansapo/response"
	"nyansapo/tokenizer"
)

type trainedCandidate struct {
	Text   string           `json:"text"`
	Vector embedding.Vector `json:"vector"`
}

type TrainedModel struct {
	Candidates []trainedCandidate `json:"candidates"`
	Knowledge  Knowledge          `json:"knowledge"`
	Facts      map[string]string  `json:"facts,omitempty"`
}

type AnswerResult struct {
	Answer     string
	Evidence   string
	Confidence float64
	Grounded   bool
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

func factTokens(question string) []string {
	tokens := tokenizer.Tokenize(question)
	sort.Strings(tokens)
	return tokens
}

func factKey(question string) string {
	return strings.Join(factTokens(question), " ")
}

func (m *TrainedModel) Teach(question, answer string) {
	question = strings.TrimSpace(question)
	answer = strings.TrimSpace(answer)
	if question == "" || answer == "" {
		return
	}
	if m.Facts == nil {
		m.Facts = make(map[string]string)
	}
	m.Facts[factKey(question)] = answer
}

func (m TrainedModel) Fact(question string) (string, bool) {
	if len(m.Facts) == 0 {
		return "", false
	}
	if answer, ok := m.Facts[factKey(question)]; ok {
		return answer, true
	}

	asked := factContentTokens(question)
	if len(asked) == 0 {
		return "", false
	}
	best := ""
	bestCoverage := 0.0
	bestSpecificity := int(^uint(0) >> 1)
	for key, answer := range m.Facts {
		required := factContentTokens(key)
		if len(required) == 0 {
			continue
		}
		coverage := tokensCoverage(asked, required)
		specificity := len(required)
		if coverage > bestCoverage ||
			(coverage == bestCoverage && specificity < bestSpecificity) {
			bestCoverage = coverage
			bestSpecificity = specificity
			best = answer
		}
	}
	if bestCoverage >= 0.6 {
		return best, true
	}
	return "", false
}

func factContentTokens(question string) []string {
	var content []string
	for _, token := range tokenizer.Tokenize(question) {
		if isNumeric(token) {
			content = append(content, token)
			continue
		}
		if len(token) >= 3 && !embedding.IsStopWord(token) {
			content = append(content, token)
		}
	}
	sort.Strings(content)
	return content
}

func isNumeric(token string) bool {
	for _, character := range token {
		if character < '0' || character > '9' {
			return false
		}
	}
	return token != ""
}

func tokensCoverage(asked, fact []string) float64 {
	counts := make(map[string]int, len(asked))
	for _, token := range asked {
		counts[token]++
	}
	matched := 0
	for _, token := range fact {
		if counts[token] > 0 {
			counts[token]--
			matched++
		}
	}
	return float64(matched) / float64(len(asked))
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
	return m.AnswerResult(question).Answer
}

func (m TrainedModel) AnswerResult(question string) AnswerResult {
	if fact, ok := m.Fact(question); ok {
		return AnswerResult{
			Answer:     fact,
			Evidence:   fact,
			Confidence: 1,
			Grounded:   true,
		}
	}

	if answer := m.Knowledge.Answer(question); strings.TrimSpace(answer) != "" &&
		answer != "I do not know that yet." {
		return AnswerResult{
			Answer:     response.Format(question, answer),
			Evidence:   answer,
			Confidence: 1,
			Grounded:   true,
		}
	}

	query := embedding.Embed(question)
	var best string
	var bestIndex int
	var bestScore float64
	for index, candidate := range m.Candidates {
		score := candidateScore(question, candidate.Text, query, candidate.Vector)
		if score > bestScore {
			bestScore = score
			best = candidate.Text
			bestIndex = index
		}
	}

	if bestScore < 0.05 {
		return AnswerResult{
			Answer:     "I do not know that yet.",
			Confidence: bestScore,
			Grounded:   false,
		}
	}

	shared := 0
	for token := range query {
		if _, ok := m.Candidates[bestIndex].Vector[token]; ok {
			shared++
		}
	}
	accepted := shared >= 2
	if shared == 1 {
		accepted = bestScore >= 0.3
	}
	if !accepted {
		return AnswerResult{
			Answer:     "I do not know that yet.",
			Confidence: bestScore,
			Grounded:   false,
		}
	}

	answer := response.Format(question, best)
	return AnswerResult{
		Answer:     answer,
		Evidence:   best,
		Confidence: bestScore,
		Grounded:   verifyEvidence(question, best, answer),
	}
}

func candidateScore(question, text string, query embedding.Vector, vector embedding.Vector) float64 {
	score := embedding.Cosine(query, vector)
	questionLower := strings.ToLower(question)
	textLower := strings.ToLower(text)
	for _, phrase := range []string{"information_schema", "window function", "primary key", "foreign key", "inner join", "left join", "common table expression", "query plan", "join", "joins"} {
		if strings.Contains(questionLower, phrase) && strings.Contains(textLower, phrase) {
			score += 1
		}
	}
	for token := range query {
		if strings.Contains(questionLower, " table") &&
			strings.HasPrefix(textLower, "table "+token+" ") {
			score += 1
		}
		if strings.Contains(questionLower, " column") &&
			strings.Contains(textLower, "columns") {
			score += 0.25
		}
		if strings.Contains(questionLower, "related") &&
			strings.Contains(textLower, "relationships") {
			score += 0.25
		}
	}
	return score
}

func verifyEvidence(question, evidence, answer string) bool {
	normalizedEvidence := strings.ToLower(strings.TrimSpace(evidence))
	normalizedAnswer := strings.ToLower(strings.TrimSpace(answer))
	if normalizedEvidence != "" && strings.Contains(normalizedAnswer, normalizedEvidence) {
		return true
	}

	queryTokens := embedding.Embed(question)
	evidenceTokens := embedding.Embed(evidence)
	answerTokens := embedding.Embed(answer)
	if len(queryTokens) == 0 || len(evidenceTokens) == 0 || len(answerTokens) == 0 {
		return false
	}

	shared := 0
	for token := range queryTokens {
		if evidenceTokens[token] > 0 {
			shared++
		}
	}
	if shared == 0 {
		return false
	}

	for token := range answerTokens {
		if evidenceTokens[token] > 0 {
			return true
		}
	}
	return false
}
