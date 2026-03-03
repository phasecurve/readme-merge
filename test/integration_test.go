package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEndToEnd(t *testing.T) {
	binPath := buildBinary(t)
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "example.go"), []byte(
		"package example\n\nfunc Add(a, b int) int {\n\treturn a + b\n}\n",
	), 0644)

	os.WriteFile(filepath.Join(dir, "README.md"), []byte(
		"# Example\n\n<!-- code from=example.go lines=3-5 -->\n<!-- /code -->\n",
	), 0644)

	cmd := exec.Command(binPath, "check")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("check should fail on unhashed placeholder")
	}
	if !strings.Contains(string(out), "unhashed") {
		t.Errorf("expected 'unhashed' message, got: %s", out)
	}

	cmd = exec.Command(binPath, "update")
	cmd.Dir = dir
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("update failed: %s", out)
	}

	readme, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	if !strings.Contains(string(readme), "func Add(a, b int) int {") {
		t.Errorf("code not injected:\n%s", readme)
	}
	if !strings.Contains(string(readme), "snippethash=") {
		t.Errorf("snippet hash not written")
	}

	cmd = exec.Command(binPath, "check")
	cmd.Dir = dir
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("check should pass after update: %s", out)
	}

	os.WriteFile(filepath.Join(dir, "example.go"), []byte(
		"package example\n\nfunc Multiply(a, b int) int {\n\treturn a * b\n}\n",
	), 0644)

	cmd = exec.Command(binPath, "check")
	cmd.Dir = dir
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatal("check should fail after source change")
	}
	if !strings.Contains(string(out), "stale") {
		t.Errorf("expected 'stale' message, got: %s", out)
	}
}

func TestEndToEndSelfHeal(t *testing.T) {
	binPath := buildBinary(t)
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "example.go"), []byte(
		"line1\ntarget code\nline3\n",
	), 0644)

	os.WriteFile(filepath.Join(dir, "README.md"), []byte(
		"# Test\n\n<!-- code from=example.go lines=2-2 -->\n<!-- /code -->\n",
	), 0644)

	cmd := exec.Command(binPath, "update")
	cmd.Dir = dir
	cmd.Run()

	os.WriteFile(filepath.Join(dir, "example.go"), []byte(
		"new first line\nline1\ntarget code\nline3\n",
	), 0644)

	cmd = exec.Command(binPath, "check")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("check should pass after self-heal: %s", out)
	}
	if !strings.Contains(string(out), "self-healed") {
		t.Errorf("expected self-heal message, got: %s", out)
	}

	readme, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	if !strings.Contains(string(readme), "lines=3-3") {
		t.Errorf("line reference not updated:\n%s", readme)
	}
}

func buildBinary(t *testing.T) string {
	t.Helper()
	binPath := filepath.Join(t.TempDir(), "readme-merge")
	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/readme-merge")
	cmd.Dir = filepath.Join(os.Getenv("HOME"), "dev", "readme-merge")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %s", out)
	}
	return binPath
}
