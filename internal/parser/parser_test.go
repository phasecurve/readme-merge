package parser_test

import (
	"strings"
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
	if b.SourceStart != 10 || b.SourceEnd != 25 {
		t.Errorf("Lines = %d-%d, want 10-25", b.SourceStart, b.SourceEnd)
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

func TestRenderNewBlock(t *testing.T) {
	original := "# Title\n\n<!-- code from=examples/client.go lines=10-12 -->\n<!-- /code -->\n\nMore text.\n"

	blocks, _ := parser.Parse(original)
	blocks[0].Content = "fmt.Println(\"hello\")\n"
	blocks[0].FileHash = "aaaa1234aaaa1234"
	blocks[0].SnippetHash = "bbbb1234bbbb1234"
	blocks[0].SourceStart = 10
	blocks[0].SourceEnd = 12

	got := parser.Render(original, blocks)

	want := "# Title\n\n<!-- code from=examples/client.go lines=10-12 filehash=aaaa1234aaaa1234 snippethash=bbbb1234bbbb1234 -->\n```go\nfmt.Println(\"hello\")\n```\n<!-- /code -->\n\nMore text.\n"
	if got != want {
		t.Errorf("Render mismatch.\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderPreservesUnchangedText(t *testing.T) {
	original := "Line 1\nLine 2\n<!-- code from=a.txt lines=1-1 -->\n<!-- /code -->\nLine 5\n"

	blocks, _ := parser.Parse(original)
	blocks[0].Content = "hello\n"
	blocks[0].FileHash = "aaaa1234aaaa1234"
	blocks[0].SnippetHash = "bbbb1234bbbb1234"

	got := parser.Render(original, blocks)

	if !strings.HasPrefix(got, "Line 1\nLine 2\n") {
		t.Errorf("text before block was modified")
	}
	if !strings.HasSuffix(got, "Line 5\n") {
		t.Errorf("text after block was modified")
	}
}
