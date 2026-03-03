package hasher_test

import (
	"testing"

	"github.com/phasecurve/readme-merge/internal/hasher"
)

func TestContentHash(t *testing.T) {
	input := "func main() {\n\tfmt.Println(\"hello\")\n}\n"
	got := hasher.ContentHash(input)

	if len(got) != 16 {
		t.Errorf("expected 16 hex chars, got %d: %q", len(got), got)
	}

	got2 := hasher.ContentHash(input)
	if got != got2 {
		t.Errorf("same input produced different hashes: %q vs %q", got, got2)
	}

	different := hasher.ContentHash("something else")
	if got == different {
		t.Errorf("different inputs produced same hash")
	}
}

func TestContentHashEmpty(t *testing.T) {
	got := hasher.ContentHash("")
	if len(got) != 16 {
		t.Errorf("expected 16 hex chars for empty string, got %d", len(got))
	}
}
