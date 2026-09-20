package api

import (
	"encoding/json"
	"net/http"

	"mini-llm/model"
)

type Server struct {
	knowledge model.Knowledge
}

func NewServer(knowledge model.Knowledge) *Server {
	return &Server{knowledge: knowledge}
}

type questionRequest struct {
	Question string `json:"question"`
}

type answerResponse struct {
	Answer string `json:"answer"`
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.URL.Path != "/ask" {
		http.NotFound(w, r)
		return
	}

	var request questionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid JSON request", http.StatusBadRequest)
		return
	}

	if request.Question == "" {
		http.Error(w, "question is required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(answerResponse{
		Answer: s.knowledge.Answer(request.Question),
	})
}
