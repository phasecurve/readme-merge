package source_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phasecurve/readme-merge/internal/source"
)

func TestWorktreeResolver(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "example.go")
	os.WriteFile(path, []byte("package main\n"), 0644)

	r := source.NewResolver("", dir)
	content, err := r.ReadFile("example.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "package main\n" {
		t.Errorf("content = %q, want %q", content, "package main\n")
	}
}

func TestWorktreeResolverMissingFile(t *testing.T) {
	dir := t.TempDir()
	r := source.NewResolver("", dir)
	_, err := r.ReadFile("nonexistent.go")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestReadFileRejectsAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	r := source.NewResolver("", dir)
	_, err := r.ReadFile("/etc/passwd")
	if err == nil {
		t.Fatal("expected error for absolute path")
	}
	if !strings.Contains(err.Error(), "absolute paths not allowed") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestReadFileRejectsParentTraversal(t *testing.T) {
	dir := t.TempDir()
	r := source.NewResolver("", dir)
	_, err := r.ReadFile("../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for parent traversal")
	}
	if !strings.Contains(err.Error(), "path escapes project directory") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestReadFileRejectsNestedTraversal(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "sub"), 0755)
	r := source.NewResolver("", dir)
	_, err := r.ReadFile("sub/../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for nested traversal")
	}
	if !strings.Contains(err.Error(), "path escapes project directory") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestReadFileAllowsSubdirectory(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "src"), 0755)
	os.WriteFile(filepath.Join(dir, "src", "main.go"), []byte("package main\n"), 0644)

	r := source.NewResolver("", dir)
	content, err := r.ReadFile("src/main.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "package main\n" {
		t.Errorf("content = %q, want %q", content, "package main\n")
	}
}

func TestReadFileAllowsDotSlashPrefix(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "file.go"), []byte("hello\n"), 0644)

	r := source.NewResolver("", dir)
	content, err := r.ReadFile("./file.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "hello\n" {
		t.Errorf("content = %q, want %q", content, "hello\n")
	}
}

func TestGitRefResolver(t *testing.T) {
	dir := t.TempDir()
	exec.Command("git", "-C", dir, "init").Run()
	exec.Command("git", "-C", dir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "Test").Run()

	path := filepath.Join(dir, "example.go")
	os.WriteFile(path, []byte("v1 content\n"), 0644)
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "init").Run()

	os.WriteFile(path, []byte("v2 content\n"), 0644)

	r := source.NewResolver("HEAD", dir)
	content, err := r.ReadFile("example.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "v1 content\n" {
		t.Errorf("HEAD content = %q, want %q", content, "v1 content\n")
	}
}
