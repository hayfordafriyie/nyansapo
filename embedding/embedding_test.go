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

func TestEmbedIgnoresQuestionStopWords(t *testing.T) {
	vector := Embed("How do plants make food?")
	if vector["how"] != 0 || vector["do"] != 0 {
		t.Fatalf("Embed() retained stop words: %#v", vector)
	}
	if vector["plants"] == 0 || vector["make"] == 0 || vector["food"] == 0 {
		t.Fatalf("Embed() removed content words: %#v", vector)
	}
}
