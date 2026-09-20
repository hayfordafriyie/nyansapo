package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"nyansapo/model"
)

type Server struct {
	answer      func(string) string
	trainedPath string
}

func NewServer(knowledge model.Knowledge) *Server {
	return &Server{answer: knowledge.Answer}
}

func NewTrainedServer(trained model.TrainedModel) *Server {
	return &Server{answer: trained.Answer}
}

func NewReloadingServer(path string) (*Server, error) {
	trained, err := model.LoadTrained(path)
	if err != nil {
		return nil, err
	}
	return &Server{answer: trained.Answer, trainedPath: path}, nil
}

type questionRequest struct {
	Question string `json:"question"`
}

type answerResponse struct {
	Answer string `json:"answer"`
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(chatPage))
		return
	}

	if r.URL.Path == "/health" {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
		return
	}

	if r.URL.Path != "/ask" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request questionRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "invalid JSON request", http.StatusBadRequest)
		return
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		http.Error(w, "request must contain one JSON object", http.StatusBadRequest)
		return
	}

	request.Question = strings.TrimSpace(request.Question)
	if request.Question == "" {
		http.Error(w, "question is required", http.StatusBadRequest)
		return
	}

	answer := s.answer
	if s.trainedPath != "" {
		trained, err := model.LoadTrained(s.trainedPath)
		if err != nil {
			http.Error(w, "trained model unavailable", http.StatusServiceUnavailable)
			return
		}
		answer = trained.Answer
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(answerResponse{
		Answer: answer(request.Question),
	}); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

const chatPage = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Nyansapo</title>
  <style>
    body { font: 16px system-ui, sans-serif; max-width: 42rem; margin: 3rem auto; padding: 0 1rem; }
    form { display: flex; gap: .5rem; }
    input { flex: 1; padding: .7rem; }
    button { padding: .7rem 1rem; }
    #answer { margin-top: 1.5rem; white-space: pre-wrap; }
  </style>
</head>
<body>
  <h1>Nyansapo</h1>
  <form id="ask-form">
    <input id="question" placeholder="Ask a question" autocomplete="off" required>
    <button>Ask</button>
  </form>
  <div id="answer" role="status"></div>
  <script>
    document.getElementById("ask-form").addEventListener("submit", async (event) => {
      event.preventDefault();
      const answer = document.getElementById("answer");
      const question = document.getElementById("question");
      answer.textContent = "Thinking...";
      const response = await fetch("/ask", {
        method: "POST",
        headers: {"Content-Type": "application/json"},
        body: JSON.stringify({question: question.value})
      });
      const data = await response.json();
      answer.textContent = response.ok ? data.answer : (data.error || "Request failed");
    });
  </script>
</body>
</html>`
