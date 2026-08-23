package booking

import (
	"strings"
	"testing"
)

func TestReference(t *testing.T) {
	first, err := reference()
	if err != nil {
		t.Fatal(err)
	}
	second, err := reference()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(first, "TIX-") {
		t.Fatalf("reference = %s", first)
	}
	if first == second {
		t.Fatal("references should be random")
	}
}
