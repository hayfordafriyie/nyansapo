package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"mini-llm/model"
)

type Server struct {
	answer func(string) string
}

func NewServer(knowledge model.Knowledge) *Server {
	return &Server{answer: knowledge.Answer}
}

func NewTrainedServer(trained model.TrainedModel) *Server {
	return &Server{answer: trained.Answer}
}

type questionRequest struct {
	Question string `json:"question"`
}

type answerResponse struct {
	Answer string `json:"answer"`
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/health" {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
		return
	}

	if r.Method != http.MethodPost || r.URL.Path != "/ask" {
		http.NotFound(w, r)
		return
	}

	var request questionRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid JSON request", http.StatusBadRequest)
		return
	}

	request.Question = strings.TrimSpace(request.Question)
	if request.Question == "" {
		http.Error(w, "question is required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(answerResponse{
		Answer: s.answer(request.Question),
	})
}
