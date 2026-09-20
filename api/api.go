package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"

	"nyansapo/database"
	"nyansapo/model"
)

type Server struct {
	answer      func(string) string
	detailed    func(string) model.AnswerResult
	data        func(context.Context, string) (database.DataAnswer, error)
	trainedPath string
	cache       *answerCache
}

func NewServer(knowledge model.Knowledge) *Server {
	return &Server{answer: knowledge.Answer, cache: newAnswerCache(defaultCacheCapacity)}
}

func NewTrainedServer(trained model.TrainedModel) *Server {
	return &Server{answer: trained.Answer, detailed: trained.AnswerResult, cache: newAnswerCache(defaultCacheCapacity)}
}

func NewDataServer(path string, assistant *database.DataAssistant) (*Server, error) {
	server, err := NewReloadingServer(path)
	if err != nil {
		return nil, err
	}
	if assistant != nil {
		server.data = assistant.Ask
	}
	return server, nil
}

func NewReloadingServer(path string) (*Server, error) {
	trained, err := model.LoadTrained(path)
	if err != nil {
		return nil, err
	}
	return &Server{answer: trained.Answer, detailed: trained.AnswerResult, trainedPath: path, cache: newAnswerCache(defaultCacheCapacity)}, nil
}

type questionRequest struct {
	Question string `json:"question"`
}

type answerResponse struct {
	Answer     string     `json:"answer"`
	Confidence float64    `json:"confidence,omitempty"`
	Grounded   bool       `json:"grounded"`
	Query      string     `json:"query,omitempty"`
	Columns    []string   `json:"columns,omitempty"`
	Rows       [][]string `json:"rows,omitempty"`
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

	cacheKey := request.Question
	if s.trainedPath != "" {
		info, err := os.Stat(s.trainedPath)
		if err != nil {
			http.Error(w, "trained model unavailable", http.StatusServiceUnavailable)
			return
		}
		cacheKey += "|" + info.ModTime().UTC().String()
	}

	// Live-data answers are paraphrased per request, so they bypass the cache
	// to keep each response naturally worded while keeping the same core facts.
	useCache := s.data == nil

	var result answerResponse
	var cached bool
	if useCache {
		result, cached = s.cache.get(cacheKey)
		if cached {
			w.Header().Set("X-Cache", "HIT")
		}
	}

	if !useCache || !cached {
		detailedFunction := s.detailed
		answerFunction := s.answer
		if s.trainedPath != "" {
			trained, err := model.LoadTrained(s.trainedPath)
			if err != nil {
				http.Error(w, "trained model unavailable", http.StatusServiceUnavailable)
				return
			}
			answerFunction = trained.Answer
			detailedFunction = trained.AnswerResult
		}
		if s.data != nil {
			dataResult, err := s.data(r.Context(), request.Question)
			if err == nil {
				result = answerResponse{
					Answer:     dataResult.Answer,
					Confidence: dataResult.Confidence,
					Grounded:   true,
					Query:      dataResult.Query,
					Columns:    dataResult.Columns,
					Rows:       dataResult.Rows,
				}
			} else if !errors.Is(err, database.ErrUnsupportedDataQuestion) {
				http.Error(w, "database query failed", http.StatusBadGateway)
				return
			}
		}
		if result.Answer == "" && database.IsSchemaQuestion(request.Question) {
			result = answerResponse{
				Answer:   "Connect a read-only business database to ask questions about live records. Nyansapo is not intended to answer schema-only questions.",
				Grounded: false,
			}
		} else if result.Answer == "" && detailedFunction != nil {
			detailed := detailedFunction(request.Question)
			result = answerResponse{
				Answer:     detailed.Answer,
				Confidence: detailed.Confidence,
				Grounded:   detailed.Grounded,
			}
		} else if result.Answer == "" {
			result = answerResponse{Answer: answerFunction(request.Question), Grounded: true}
		}
		if useCache {
			s.cache.set(cacheKey, result)
			w.Header().Set("X-Cache", "MISS")
		} else {
			w.Header().Set("X-Cache", "BYPASS")
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
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
