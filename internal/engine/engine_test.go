package engine_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phasecurve/readme-merge/internal/engine"
	"github.com/phasecurve/readme-merge/internal/source"
)

type stubReader struct {
	files map[string]string
}

func (s *stubReader) ReadFile(path string) (string, error) {
	content, ok := s.files[path]
	if !ok {
		return "", fmt.Errorf("file not found: %s", path)
	}
	return content, nil
}

func TestUpdateNewPlaceholder(t *testing.T) {
	reader := &stubReader{files: map[string]string{
		"src/main.go": "package main\n\nfunc hello() {\n\treturn\n}\n",
	}}
	content := "# Proj\n\n<!-- code from=src/main.go lines=3-5 -->\n<!-- /code -->\n"

	result, err := engine.Update(content, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.Output, "func hello()") {
		t.Errorf("expected code to be injected, got:\n%s", result.Output)
	}
	if !strings.Contains(result.Output, "snippethash=") {
		t.Errorf("expected snippet hash in output")
	}
	if result.Updated != 1 {
		t.Errorf("Updated = %d, want 1", result.Updated)
	}
}

func TestUpdateRejectsOutOfBoundsLines(t *testing.T) {
	reader := &stubReader{files: map[string]string{
		"src/main.go": "line1\nline2\n",
	}}
	content := "<!-- code from=src/main.go lines=1-99 -->\n<!-- /code -->\n"

	_, err := engine.Update(content, reader)
	if err == nil {
		t.Fatal("expected error for out-of-bounds line range")
	}
	if !strings.Contains(err.Error(), "out of bounds") {
		t.Errorf("expected 'out of bounds' error, got: %v", err)
	}
}

func TestUpdateRejectsPathTraversal(t *testing.T) {
	parent := t.TempDir()
	os.WriteFile(filepath.Join(parent, "secret.txt"), []byte("leaked\n"), 0644)

	dir := filepath.Join(parent, "proj")
	os.MkdirAll(dir, 0755)

	resolver := source.NewResolver("", dir)
	content := "<!-- code from=../secret.txt lines=1-1 -->\n<!-- /code -->\n"

	_, err := engine.Update(content, resolver)
	if err == nil {
		t.Fatal("expected error for path traversal through engine")
	}
	if !strings.Contains(err.Error(), "escapes project directory") {
		t.Errorf("expected path escape error, got: %v", err)
	}
}

func TestCheckStaleSnippet(t *testing.T) {
	reader := &stubReader{files: map[string]string{
		"src/main.go": "package main\n\nfunc hello() {\n\treturn\n}\n",
	}}
	content := "# Proj\n\n<!-- code from=src/main.go lines=3-5 -->\n<!-- /code -->\n"

	updated, err := engine.Update(content, reader)
	if err != nil {
		t.Fatalf("update error: %v", err)
	}

	reader.files["src/main.go"] = "package main\n\nfunc goodbye() {\n\treturn\n}\n"

	result, err := engine.Check(updated.Output, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Stale) != 1 {
		t.Errorf("expected 1 stale block, got %d", len(result.Stale))
	}
}

func TestCheckSelfHealing(t *testing.T) {
	reader := &stubReader{files: map[string]string{
		"src/main.go": "line1\nline2\ntarget\nline4\n",
	}}
	content := "<!-- code from=src/main.go lines=3-3 -->\n<!-- /code -->\n"

	updated, err := engine.Update(content, reader)
	if err != nil {
		t.Fatalf("update error: %v", err)
	}

	reader.files["src/main.go"] = "new top\nline1\nline2\ntarget\nline4\n"

	result, err := engine.Check(updated.Output, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Healed != 1 {
		t.Errorf("Healed = %d, want 1", result.Healed)
	}
	if len(result.Stale) != 0 {
		t.Errorf("expected no stale blocks after heal")
	}
	if !strings.Contains(result.Output, "lines=4-4") {
		t.Errorf("expected healed line reference, got:\n%s", result.Output)
	}
}

func TestCheckHealOutputIsEmpty(t *testing.T) {
	reader := &stubReader{files: map[string]string{
		"src/main.go": "line1\nline2\ntarget\nline4\n",
	}}
	content := "<!-- code from=src/main.go lines=3-3 -->\n<!-- /code -->\n"

	updated, err := engine.Update(content, reader)
	if err != nil {
		t.Fatalf("update error: %v", err)
	}

	result, err := engine.Check(updated.Output, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Output != "" {
		t.Errorf("expected empty Output when no changes needed, got:\n%s", result.Output)
	}
}
