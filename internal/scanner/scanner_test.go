package scanner_test

import (
	"testing"

	"github.com/phasecurve/readme-merge/internal/hasher"
	"github.com/phasecurve/readme-merge/internal/scanner"
)

func TestFindRelocatedSnippet(t *testing.T) {
	snippet := "target line 1\ntarget line 2\n"
	snippetHash := hasher.ContentHash(snippet)

	fileContent := "new line\nnew line 2\ntarget line 1\ntarget line 2\nmore stuff\n"

	start, end, found := scanner.FindSnippet(fileContent, snippetHash, 2)
	if !found {
		t.Fatal("expected to find relocated snippet")
	}
	if start != 3 || end != 4 {
		t.Errorf("lines = %d-%d, want 3-4", start, end)
	}
}

func TestSnippetNotFound(t *testing.T) {
	snippetHash := hasher.ContentHash("this content is gone\n")
	fileContent := "totally different\ncontent here\n"

	_, _, found := scanner.FindSnippet(fileContent, snippetHash, 1)
	if found {
		t.Fatal("should not find snippet that doesn't exist")
	}
}

func TestSnippetAtOriginalPosition(t *testing.T) {
	snippet := "line A\nline B\nline C\n"
	snippetHash := hasher.ContentHash(snippet)

	fileContent := "line A\nline B\nline C\nline D\n"

	start, end, found := scanner.FindSnippet(fileContent, snippetHash, 3)
	if !found {
		t.Fatal("expected to find snippet at original position")
	}
	if start != 1 || end != 3 {
		t.Errorf("lines = %d-%d, want 1-3", start, end)
	}
}
