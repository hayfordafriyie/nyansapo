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
