package tokenizer

import (
	"reflect"
	"testing"
)

func TestTokenize(t *testing.T) {
	got := Tokenize("What is EDSPiKE?")
	want := []string{"what", "is", "edspike"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Tokenize() = %#v, want %#v", got, want)
	}
}
