package evaluation

import (
	"net/http/httptest"
	"testing"

	"mini-llm/api"
	"mini-llm/model"
)

func TestRunAPI(t *testing.T) {
	server := httptest.NewServer(api.NewTrainedServer(model.TrainTexts([]string{
		"Photosynthesis converts light energy into chemical energy.",
		"A fraction represents a part of a whole.",
	})))
	defer server.Close()

	result, err := RunAPI(server.Client(), server.URL+"/ask", 1)
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != len(Questions) {
		t.Fatalf("Total = %d, want %d", result.Total, len(Questions))
	}
}
