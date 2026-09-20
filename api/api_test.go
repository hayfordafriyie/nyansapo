package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mini-llm/model"
)

func TestAsk(t *testing.T) {
	server := NewServer(model.Knowledge{
		Description: "A platform.",
	})
	request := httptest.NewRequest(http.MethodPost, "/ask", strings.NewReader(`{"question":"What is EDSPiKE?"}`))
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), "A platform.") {
		t.Fatalf("response = %q, want answer", response.Body.String())
	}
}

func TestAskRejectsMissingQuestion(t *testing.T) {
	server := NewServer(model.Knowledge{})
	request := httptest.NewRequest(http.MethodPost, "/ask", strings.NewReader(`{}`))
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestAskRejectsMalformedJSON(t *testing.T) {
	server := NewServer(model.Knowledge{})
	request := httptest.NewRequest(http.MethodPost, "/ask", strings.NewReader(`{"question":`))
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestAskRejectsTrailingJSON(t *testing.T) {
	server := NewServer(model.Knowledge{Description: "A platform."})
	request := httptest.NewRequest(http.MethodPost, "/ask", strings.NewReader(`{"question":"test"} {}`))
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestAskRejectsUnsupportedMethod(t *testing.T) {
	server := NewServer(model.Knowledge{})
	request := httptest.NewRequest(http.MethodGet, "/ask", nil)
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
}

func TestHealth(t *testing.T) {
	server := NewServer(model.Knowledge{})
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.String() != `{"status":"ok"}` {
		t.Fatalf("response = %q, want health response", response.Body.String())
	}
}

func TestChatPage(t *testing.T) {
	server := NewServer(model.Knowledge{})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("content type = %q, want HTML", response.Header().Get("Content-Type"))
	}
	if !strings.Contains(response.Body.String(), "EDSPiKE Assistant") {
		t.Fatalf("response does not contain chat page title")
	}
}

func TestReloadingServer(t *testing.T) {
	path := t.TempDir() + "/model.json"
	trained := model.Train(model.Knowledge{
		Name:        "EDSPiKE",
		Country:     "Ghana",
		Description: "A platform.",
	})
	if err := trained.Save(path); err != nil {
		t.Fatal(err)
	}
	server, err := NewReloadingServer(path)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/ask", strings.NewReader(`{"question":"What country?"}`))
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Ghana") {
		t.Fatalf("status = %d, response = %q", response.Code, response.Body.String())
	}
}
