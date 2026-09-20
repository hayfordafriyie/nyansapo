package embedding

import "testing"

func TestCosineRanksSharedWords(t *testing.T) {
	query := Embed("school management")
	match := Embed("School management")
	other := Embed("teacher management")

	if Cosine(query, match) <= Cosine(query, other) {
		t.Fatalf("expected school match to rank higher")
	}
}
