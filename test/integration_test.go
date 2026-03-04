package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitCheckout(t *testing.T, paths ...string) {
	t.Helper()
	args := append([]string{"checkout"}, paths...)
	cmd := exec.Command("git", args...)
	cmd.Dir = projectRoot()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git checkout failed: %s", out)
	}
}

func projectRoot() string {
	return filepath.Join(os.Getenv("HOME"), "dev", "readme-merge")
}

func testdataDir(name string) string {
	return filepath.Join(projectRoot(), "test", "testdata", name)
}

func TestEndToEnd(t *testing.T) {
	binPath := buildBinary(t)
	dir := testdataDir("basic")

	t.Cleanup(func() {
		gitCheckout(t, "test/testdata/basic/")
	})

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
	dir := testdataDir("selfheal")

	t.Cleanup(func() {
		gitCheckout(t, "test/testdata/selfheal/")
	})

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

func TestEndToEndRejectsPathTraversal(t *testing.T) {
	binPath := buildBinary(t)
	parent := t.TempDir()

	secret := filepath.Join(parent, "secret.txt")
	os.WriteFile(secret, []byte("do not leak this\n"), 0644)

	project := filepath.Join(parent, "myproject")
	os.MkdirAll(project, 0755)

	os.WriteFile(filepath.Join(project, "legit.go"), []byte(
		"package legit\n\nfunc Hello() string {\n\treturn \"hi\"\n}\n",
	), 0644)

	os.WriteFile(filepath.Join(project, "README.md"), []byte(
		"# My Project\n\n<!-- code from=../secret.txt lines=1-1 -->\n<!-- /code -->\n",
	), 0644)

	cmd := exec.Command(binPath, "update")
	cmd.Dir = project
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("update should fail when from= traverses outside project")
	}
	if !strings.Contains(string(out), "escapes project directory") {
		t.Errorf("expected 'escapes project directory' error, got: %s", out)
	}

	readme, _ := os.ReadFile(filepath.Join(project, "README.md"))
	if strings.Contains(string(readme), "do not leak") {
		t.Fatal("secret content was leaked into README")
	}
}

func TestEndToEndRejectsAbsolutePathInFrom(t *testing.T) {
	binPath := buildBinary(t)
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "README.md"), []byte(
		"# Docs\n\n<!-- code from=/etc/hostname lines=1-1 -->\n<!-- /code -->\n",
	), 0644)

	cmd := exec.Command(binPath, "update")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("update should fail when from= uses absolute path")
	}
	if !strings.Contains(string(out), "absolute paths not allowed") {
		t.Errorf("expected 'absolute paths not allowed' error, got: %s", out)
	}
}

func TestEndToEndAllowsSubdirectoryFrom(t *testing.T) {
	binPath := buildBinary(t)
	dir := testdataDir("subdirectory")

	t.Cleanup(func() {
		gitCheckout(t, "test/testdata/subdirectory/")
	})

	cmd := exec.Command(binPath, "update")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("update should succeed for subdirectory path: %s", out)
	}

	readme, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	if !strings.Contains(string(readme), "fmt.Println") {
		t.Errorf("code from subdirectory not injected:\n%s", readme)
	}
}

func buildBinary(t *testing.T) string {
	t.Helper()
	binPath := filepath.Join(t.TempDir(), "readme-merge")
	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/readme-merge")
	cmd.Dir = projectRoot()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %s", out)
	}
	return binPath
}
