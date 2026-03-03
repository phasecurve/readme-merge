package engine_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phasecurve/readme-merge/internal/engine"
	"github.com/phasecurve/readme-merge/internal/source"
)

func setupProject(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		os.MkdirAll(filepath.Dir(path), 0755)
		os.WriteFile(path, []byte(content), 0644)
	}
	return dir
}

func TestUpdateNewPlaceholder(t *testing.T) {
	dir := setupProject(t, map[string]string{
		"src/main.go": "package main\n\nfunc hello() {\n\treturn\n}\n",
		"README.md":   "# Proj\n\n<!-- code from=src/main.go lines=3-5 -->\n<!-- /code -->\n",
	})

	r := source.NewResolver("", dir)
	result, err := engine.Update(filepath.Join(dir, "README.md"), r)
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

func TestCheckStaleSnippet(t *testing.T) {
	dir := setupProject(t, map[string]string{
		"src/main.go": "package main\n\nfunc hello() {\n\treturn\n}\n",
		"README.md":   "# Proj\n\n<!-- code from=src/main.go lines=3-5 -->\n<!-- /code -->\n",
	})

	r := source.NewResolver("", dir)
	engine.Update(filepath.Join(dir, "README.md"), r)

	os.WriteFile(filepath.Join(dir, "src/main.go"),
		[]byte("package main\n\nfunc goodbye() {\n\treturn\n}\n"), 0644)

	result, err := engine.Check(filepath.Join(dir, "README.md"), r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Stale) != 1 {
		t.Errorf("expected 1 stale block, got %d", len(result.Stale))
	}
}

func TestCheckSelfHealing(t *testing.T) {
	dir := setupProject(t, map[string]string{
		"src/main.go": "line1\nline2\ntarget\nline4\n",
		"README.md":   "<!-- code from=src/main.go lines=3-3 -->\n<!-- /code -->\n",
	})

	r := source.NewResolver("", dir)
	engine.Update(filepath.Join(dir, "README.md"), r)

	os.WriteFile(filepath.Join(dir, "src/main.go"),
		[]byte("new top\nline1\nline2\ntarget\nline4\n"), 0644)

	result, err := engine.Check(filepath.Join(dir, "README.md"), r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Healed != 1 {
		t.Errorf("Healed = %d, want 1", result.Healed)
	}
	if len(result.Stale) != 0 {
		t.Errorf("expected no stale blocks after heal")
	}

	data, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	if !strings.Contains(string(data), "lines=4-4") {
		t.Errorf("expected healed line reference, got:\n%s", string(data))
	}
}
