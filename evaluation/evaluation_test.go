package evaluation

import (
	"testing"

	"nyansapo/model"
)

func TestRunCountsQueries(t *testing.T) {
	result := Run(model.TrainTexts([]string{
		"Photosynthesis converts light energy into chemical energy.",
		"A prime number has exactly two positive factors.",
	}), 2)
	if result.Total != 40 {
		t.Fatalf("Total = %d, want 40", result.Total)
	}
}
