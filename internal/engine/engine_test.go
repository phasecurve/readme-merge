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

func (s *stubReader) ReadFile(path string, ref string) (string, error) {
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

func TestCheckReturnsFreshBlocks(t *testing.T) {
	reader := &stubReader{files: map[string]string{
		"src/main.go": "package main\n\nfunc hello() {\n\treturn\n}\n",
	}}
	content := "<!-- code from=src/main.go lines=3-5 -->\n<!-- /code -->\n"

	updated, err := engine.Update(content, reader)
	if err != nil {
		t.Fatalf("update error: %v", err)
	}

	result, err := engine.Check(updated.Output, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.FreshBlocks) != 1 {
		t.Fatalf("expected 1 fresh block, got %d", len(result.FreshBlocks))
	}
	b := result.FreshBlocks[0]
	if b.From != "src/main.go" {
		t.Errorf("From = %q, want %q", b.From, "src/main.go")
	}
	if b.Content == "" {
		t.Error("expected fresh block to have content")
	}
}

func TestUpdateIslandSingleRange(t *testing.T) {
	reader := &stubReader{files: map[string]string{
		"docs/guide.md": "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\nline11\nline12\nline13\nline14\n",
	}}
	content := "# Proj\n\n<!-- island file=\"docs/guide.md\" -->\n<!-- lines from=\"10\" to=\"14\" -->\n<!-- end island -->\n"

	result, err := engine.Update(content, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.Output, "line10") {
		t.Errorf("expected island content, got:\n%s", result.Output)
	}
	if strings.Contains(result.Output, "```") {
		t.Errorf("island should render raw (no fences), got:\n%s", result.Output)
	}
	if !strings.Contains(result.Output, "snippethash=") {
		t.Errorf("expected snippet hash in output")
	}
	if result.Updated != 1 {
		t.Errorf("Updated = %d, want 1", result.Updated)
	}
}

func TestUpdateIslandMultipleRanges(t *testing.T) {
	lines := make([]string, 65)
	for i := range lines {
		lines[i] = fmt.Sprintf("line%d", i+1)
	}
	reader := &stubReader{files: map[string]string{
		"docs/guide.md": strings.Join(lines, "\n") + "\n",
	}}
	content := "<!-- island file=\"docs/guide.md\" -->\n<!-- lines from=\"10\" to=\"14\" -->\n<!-- lines from=\"54\" to=\"62\" -->\n<!-- end island -->\n"

	result, err := engine.Update(content, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.Output, "line10") {
		t.Errorf("expected first range content, got:\n%s", result.Output)
	}
	if !strings.Contains(result.Output, "line54") {
		t.Errorf("expected second range content, got:\n%s", result.Output)
	}
	if result.Updated != 2 {
		t.Errorf("Updated = %d, want 2", result.Updated)
	}
}

func TestCheckIslandStale(t *testing.T) {
	reader := &stubReader{files: map[string]string{
		"docs/guide.md": "line1\nline2\nline3\nline4\nline5\n",
	}}
	content := "<!-- island file=\"docs/guide.md\" -->\n<!-- lines from=\"3\" to=\"5\" -->\n<!-- end island -->\n"

	updated, err := engine.Update(content, reader)
	if err != nil {
		t.Fatalf("update error: %v", err)
	}

	reader.files["docs/guide.md"] = "line1\nline2\nCHANGED\nline4\nline5\n"

	result, err := engine.Check(updated.Output, reader)
	if err != nil {
		t.Fatalf("check error: %v", err)
	}
	if len(result.Stale) != 1 {
		t.Errorf("expected 1 stale block, got %d", len(result.Stale))
	}
}

func TestCheckIslandSelfHealing(t *testing.T) {
	reader := &stubReader{files: map[string]string{
		"docs/guide.md": "line1\nline2\ntarget\nline4\n",
	}}
	content := "<!-- island file=\"docs/guide.md\" -->\n<!-- lines from=\"3\" to=\"3\" -->\n<!-- end island -->\n"

	updated, err := engine.Update(content, reader)
	if err != nil {
		t.Fatalf("update error: %v", err)
	}

	reader.files["docs/guide.md"] = "new top\nline1\nline2\ntarget\nline4\n"

	result, err := engine.Check(updated.Output, reader)
	if err != nil {
		t.Fatalf("check error: %v", err)
	}
	if result.Healed != 1 {
		t.Errorf("Healed = %d, want 1", result.Healed)
	}
	if len(result.Stale) != 0 {
		t.Errorf("expected no stale blocks after heal")
	}
	if !strings.Contains(result.Output, "from=\"4\"") {
		t.Errorf("expected healed line reference, got:\n%s", result.Output)
	}
}

func TestUpdateIslandRoundTrip(t *testing.T) {
	lines := make([]string, 65)
	for i := range lines {
		lines[i] = fmt.Sprintf("line%d", i+1)
	}
	reader := &stubReader{files: map[string]string{
		"docs/guide.md": strings.Join(lines, "\n") + "\n",
	}}
	content := "<!-- island file=\"docs/guide.md\" -->\n<!-- lines from=\"10\" to=\"12\" -->\n<!-- lines from=\"54\" to=\"56\" -->\n<!-- end island -->\n"

	first, err := engine.Update(content, reader)
	if err != nil {
		t.Fatalf("first update error: %v", err)
	}

	second, err := engine.Update(first.Output, reader)
	if err != nil {
		t.Fatalf("second update error: %v", err)
	}

	if first.Output != second.Output {
		t.Errorf("round-trip not stable.\nfirst:\n%s\nsecond:\n%s", first.Output, second.Output)
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
