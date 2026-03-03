package parser_test

import (
	"testing"

	"github.com/phasecurve/readme-merge/internal/parser"
)

func TestParseNewPlaceholder(t *testing.T) {
	input := "# Title\n\n<!-- code from=examples/client.go lines=10-25 -->\n<!-- /code -->\n\nMore text.\n"

	blocks, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	b := blocks[0]
	if b.From != "examples/client.go" {
		t.Errorf("From = %q, want %q", b.From, "examples/client.go")
	}
	if b.LineStart != 10 || b.LineEnd != 25 {
		t.Errorf("Lines = %d-%d, want 10-25", b.LineStart, b.LineEnd)
	}
	if b.FileHash != "" || b.SnippetHash != "" {
		t.Errorf("new placeholder should have empty hashes")
	}
	if b.Content != "" {
		t.Errorf("new placeholder should have empty content")
	}
}

func TestParsePopulatedPlaceholder(t *testing.T) {
	input := "<!-- code from=src/main.go lines=1-3 filehash=aaaa5678 snippethash=bbbb5678 -->\n```go\npackage main\n```\n<!-- /code -->\n"

	blocks, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	b := blocks[0]
	if b.FileHash != "aaaa5678" {
		t.Errorf("FileHash = %q, want %q", b.FileHash, "aaaa5678")
	}
	if b.SnippetHash != "bbbb5678" {
		t.Errorf("SnippetHash = %q, want %q", b.SnippetHash, "bbbb5678")
	}
	if b.Content != "package main\n" {
		t.Errorf("Content = %q, want %q", b.Content, "package main\n")
	}
}

func TestParseMultiplePlaceholders(t *testing.T) {
	input := "<!-- code from=a.go lines=1-2 -->\n<!-- /code -->\n\nText\n\n<!-- code from=b.go lines=5-10 -->\n<!-- /code -->\n"

	blocks, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
	if blocks[0].From != "a.go" || blocks[1].From != "b.go" {
		t.Errorf("wrong file refs: %q, %q", blocks[0].From, blocks[1].From)
	}
}
