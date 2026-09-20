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
