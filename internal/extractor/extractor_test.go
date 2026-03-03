package extractor_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/phasecurve/readme-merge/internal/extractor"
)

func TestExtractLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "example.go")
	content := "line1\nline2\nline3\nline4\nline5\n"
	os.WriteFile(path, []byte(content), 0644)

	snippet, fileContent, err := extractor.Extract(path, 2, 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snippet != "line2\nline3\nline4\n" {
		t.Errorf("snippet = %q, want %q", snippet, "line2\nline3\nline4\n")
	}
	if fileContent != content {
		t.Errorf("fileContent = %q, want %q", fileContent, content)
	}
}

func TestExtractOutOfRange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "small.go")
	os.WriteFile(path, []byte("line1\nline2\n"), 0644)

	_, _, err := extractor.Extract(path, 1, 10)
	if err == nil {
		t.Fatal("expected error for out-of-range lines")
	}
}

func TestExtractFileNotFound(t *testing.T) {
	_, _, err := extractor.Extract("/nonexistent/file.go", 1, 5)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
