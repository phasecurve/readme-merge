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
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return filepath.Dir(wd)
}

func testdataDir(name string) string {
	return filepath.Join(projectRoot(), "test", "testdata", name)
}

func runUpdate(t *testing.T, binPath, dir string) {
	t.Helper()
	cmd := exec.Command(binPath, "update")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("update failed: %s", out)
	}
}

func runCheck(t *testing.T, binPath, dir string) (string, error) {
	t.Helper()
	cmd := exec.Command(binPath, "check")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func runCheckWithHeal(t *testing.T, binPath, dir string) (string, error) {
	t.Helper()
	cmd := exec.Command(binPath, "check", "--heal")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestEndToEnd(t *testing.T) {
	binPath := buildBinary(t)
	dir := testdataDir("basic")

	t.Cleanup(func() {
		gitCheckout(t, "test/testdata/basic/")
	})

	out, err := runCheck(t, binPath, dir)
	if err == nil {
		t.Fatal("check should fail on unhashed placeholder")
	}
	if !strings.Contains(out, "unhashed") {
		t.Errorf("expected 'unhashed' message, got: %s", out)
	}

	runUpdate(t, binPath, dir)

	readme, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	if !strings.Contains(string(readme), "func Add(a, b int) int {") {
		t.Errorf("code not injected:\n%s", readme)
	}
	if !strings.Contains(string(readme), "snippethash=") {
		t.Errorf("snippet hash not written")
	}

	out, err = runCheck(t, binPath, dir)
	if err != nil {
		t.Fatalf("check should pass after update: %s", out)
	}

	os.WriteFile(filepath.Join(dir, "example.go"), []byte(
		"package example\n\nfunc Multiply(a, b int) int {\n\treturn a * b\n}\n",
	), 0644)

	out, err = runCheck(t, binPath, dir)
	if err == nil {
		t.Fatal("check should fail after source change")
	}
	if !strings.Contains(out, "stale") {
		t.Errorf("expected 'stale' message, got: %s", out)
	}
}

func TestHealAddSingle(t *testing.T) {
	binPath := buildBinary(t)
	dir := testdataDir("selfheal")

	t.Cleanup(func() {
		gitCheckout(t, "test/testdata/selfheal/")
	})

	runUpdate(t, binPath, dir)

	os.WriteFile(filepath.Join(dir, "example.go"), []byte(
		"package example\n\nvar line1 = \"line1\"\n\n// new comment\nfunc Target() string {\n\treturn \"target code\"\n}\n\nvar line3 = \"line3\"\n",
	), 0644)

	out, err := runCheckWithHeal(t, binPath, dir)
	if err != nil {
		t.Fatalf("check --heal should pass after self-heal: %s", out)
	}
	if !strings.Contains(out, "self-healed") {
		t.Errorf("expected self-heal message, got: %s", out)
	}

	readme, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	if !strings.Contains(string(readme), "lines=6-8") {
		t.Errorf("line reference not updated:\n%s", readme)
	}
}

func TestUpdateHonoursSnippetHash(t *testing.T) {
	binPath := buildBinary(t)
	dir := testdataDir("selfheal")

	t.Cleanup(func() {
		gitCheckout(t, "test/testdata/selfheal/")
	})

	runUpdate(t, binPath, dir)

	readme, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	if !strings.Contains(string(readme), "return \"target code\"") {
		t.Fatalf("first update should inject Target func:\n%s", readme)
	}

	os.WriteFile(filepath.Join(dir, "example.go"), []byte(
		"package example\n\nvar line1 = \"line1\"\n\n// new comment\nfunc Target() string {\n\treturn \"target code\"\n}\n\nvar line3 = \"line3\"\n",
	), 0644)

	runUpdate(t, binPath, dir)

	readme, _ = os.ReadFile(filepath.Join(dir, "README.md"))
	if !strings.Contains(string(readme), "return \"target code\"") {
		t.Errorf("second update should still contain Target func (not content from old line position):\n%s", readme)
	}
	if !strings.Contains(string(readme), "lines=6-8") {
		t.Errorf("second update should heal line reference to 6-8:\n%s", readme)
	}
}

func TestHealAddMultiple(t *testing.T) {
	binPath := buildBinary(t)
	dir := testdataDir("heal-add-multiple")

	t.Cleanup(func() {
		gitCheckout(t, "test/testdata/heal-add-multiple/")
	})

	runUpdate(t, binPath, dir)

	os.WriteFile(filepath.Join(dir, "example.go"), []byte(
		"package example\n\n// added one\n// added two\n// added three\nfunc Hello() {\n\tfmt.Println(\"hello world\")\n}\n",
	), 0644)

	out, err := runCheckWithHeal(t, binPath, dir)
	if err != nil {
		t.Fatalf("check --heal should pass after self-heal: %s", out)
	}
	if !strings.Contains(out, "self-healed") {
		t.Errorf("expected self-heal message, got: %s", out)
	}

	readme, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	if !strings.Contains(string(readme), "lines=6-8") {
		t.Errorf("line reference not updated to 6-8:\n%s", readme)
	}
}

func TestHealRemoveSingle(t *testing.T) {
	binPath := buildBinary(t)
	dir := testdataDir("heal-remove-single")

	t.Cleanup(func() {
		gitCheckout(t, "test/testdata/heal-remove-single/")
	})

	runUpdate(t, binPath, dir)

	os.WriteFile(filepath.Join(dir, "example.go"), []byte(
		"package example\n\nfunc Hello() {\n\tfmt.Println(\"hello world\")\n}\n",
	), 0644)

	out, err := runCheckWithHeal(t, binPath, dir)
	if err != nil {
		t.Fatalf("check --heal should pass after self-heal: %s", out)
	}
	if !strings.Contains(out, "self-healed") {
		t.Errorf("expected self-heal message, got: %s", out)
	}

	readme, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	if !strings.Contains(string(readme), "lines=3-5") {
		t.Errorf("line reference not updated to 3-5:\n%s", readme)
	}
}

func TestHealRemoveMultiple(t *testing.T) {
	binPath := buildBinary(t)
	dir := testdataDir("heal-remove-multiple")

	t.Cleanup(func() {
		gitCheckout(t, "test/testdata/heal-remove-multiple/")
	})

	runUpdate(t, binPath, dir)

	os.WriteFile(filepath.Join(dir, "example.go"), []byte(
		"package example\n\nfunc Hello() {\n\tfmt.Println(\"hello world\")\n}\n",
	), 0644)

	out, err := runCheckWithHeal(t, binPath, dir)
	if err != nil {
		t.Fatalf("check --heal should pass after self-heal: %s", out)
	}
	if !strings.Contains(out, "self-healed") {
		t.Errorf("expected self-heal message, got: %s", out)
	}

	readme, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	if !strings.Contains(string(readme), "lines=3-5") {
		t.Errorf("line reference not updated to 3-5:\n%s", readme)
	}
}

func TestContentChangeShouldReportStale(t *testing.T) {
	binPath := buildBinary(t)
	dir := testdataDir("content-change")

	t.Cleanup(func() {
		gitCheckout(t, "test/testdata/content-change/")
	})

	runUpdate(t, binPath, dir)

	os.WriteFile(filepath.Join(dir, "example.go"), []byte(
		"package example\n\n// added line\nfunc Goodbye() {\n\tfmt.Println(\"goodbye world\")\n}\n",
	), 0644)

	out, err := runCheck(t, binPath, dir)
	if err == nil {
		t.Fatal("check should fail when content genuinely changed (even if shifted)")
	}
	if !strings.Contains(out, "stale") {
		t.Errorf("expected 'stale' message, got: %s", out)
	}
}

func TestPlaceholderAddSingleAbove(t *testing.T) {
	binPath := buildBinary(t)
	dir := testdataDir("multi-placeholder")

	t.Cleanup(func() {
		gitCheckout(t, "test/testdata/multi-placeholder/")
	})

	runUpdate(t, binPath, dir)

	readme, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	original := string(readme)
	if !strings.Contains(original, "snippethash=") {
		t.Fatal("update should have written snippet hashes")
	}

	newPlaceholder := "## New Section\n\n<!-- code from=funcs.go lines=3-5 -->\n<!-- /code -->\n\n"
	modified := strings.Replace(original, "## Beta", newPlaceholder+"## Beta", 1)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte(modified), 0644)

	out, err := runCheck(t, binPath, dir)
	if err == nil {
		t.Fatal("check should fail due to unhashed new placeholder")
	}
	if !strings.Contains(out, "unhashed") {
		t.Errorf("expected 'unhashed' for new placeholder, got: %s", out)
	}
	if !strings.Contains(out, "fresh") {
		t.Errorf("existing placeholders should still be fresh, got: %s", out)
	}
}

func TestPlaceholderAddMultipleAbove(t *testing.T) {
	binPath := buildBinary(t)
	dir := testdataDir("multi-placeholder")

	t.Cleanup(func() {
		gitCheckout(t, "test/testdata/multi-placeholder/")
	})

	runUpdate(t, binPath, dir)

	readme, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	original := string(readme)

	newPlaceholders := "## New One\n\n<!-- code from=funcs.go lines=3-5 -->\n<!-- /code -->\n\n" +
		"## New Two\n\n<!-- code from=funcs.go lines=7-9 -->\n<!-- /code -->\n\n"
	modified := strings.Replace(original, "## Gamma", newPlaceholders+"## Gamma", 1)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte(modified), 0644)

	out, err := runCheck(t, binPath, dir)
	if err == nil {
		t.Fatal("check should fail due to unhashed new placeholders")
	}
	if !strings.Contains(out, "unhashed") {
		t.Errorf("expected 'unhashed' for new placeholders, got: %s", out)
	}
}

func TestPlaceholderRemoveSingleAbove(t *testing.T) {
	binPath := buildBinary(t)
	dir := testdataDir("multi-placeholder")

	t.Cleanup(func() {
		gitCheckout(t, "test/testdata/multi-placeholder/")
	})

	runUpdate(t, binPath, dir)

	readme, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	lines := strings.Split(string(readme), "\n")

	var filtered []string
	alphaStart := -1
	alphaEnd := -1
	for i, line := range lines {
		if strings.Contains(line, "## Alpha") {
			alphaStart = i
		}
		if alphaStart >= 0 && alphaEnd < 0 && strings.Contains(line, "<!-- /code -->") {
			alphaEnd = i
		}
	}
	filtered = append(filtered, lines[:alphaStart]...)
	filtered = append(filtered, lines[alphaEnd+1:]...)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte(strings.Join(filtered, "\n")), 0644)

	out, err := runCheck(t, binPath, dir)
	if err != nil {
		t.Fatalf("check should pass with remaining placeholders fresh: %s", out)
	}
	if !strings.Contains(out, "fresh") {
		t.Errorf("remaining placeholders should be fresh, got: %s", out)
	}
}

func TestPlaceholderRemoveMultipleAbove(t *testing.T) {
	binPath := buildBinary(t)
	dir := testdataDir("multi-placeholder")

	t.Cleanup(func() {
		gitCheckout(t, "test/testdata/multi-placeholder/")
	})

	runUpdate(t, binPath, dir)

	readme, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	lines := strings.Split(string(readme), "\n")

	gammaStart := -1
	for i, line := range lines {
		if strings.Contains(line, "## Gamma") {
			gammaStart = i
			break
		}
	}
	remaining := append(lines[:1], lines[gammaStart:]...)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte(strings.Join(remaining, "\n")), 0644)

	out, err := runCheck(t, binPath, dir)
	if err != nil {
		t.Fatalf("check should pass with Gamma placeholder fresh: %s", out)
	}
	if !strings.Contains(out, "fresh") {
		t.Errorf("remaining placeholder should be fresh, got: %s", out)
	}
}

func TestDeepSubdirectoryFrom(t *testing.T) {
	binPath := buildBinary(t)
	dir := testdataDir("deep-subdir")

	t.Cleanup(func() {
		gitCheckout(t, "test/testdata/deep-subdir/")
	})

	runUpdate(t, binPath, dir)

	readme, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	if !strings.Contains(string(readme), "fmt.Println(\"deeply nested\")") {
		t.Errorf("code from deep subdir not injected:\n%s", readme)
	}

	out, err := runCheck(t, binPath, dir)
	if err != nil {
		t.Fatalf("check should pass for deep subdir: %s", out)
	}
}

func TestRejectsPathTraversal(t *testing.T) {
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

func TestRejectsAbsolutePathInFrom(t *testing.T) {
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

func TestRejectsPathUpMultipleLevels(t *testing.T) {
	binPath := buildBinary(t)
	parent := t.TempDir()

	secret := filepath.Join(parent, "secret.txt")
	os.WriteFile(secret, []byte("top secret\n"), 0644)

	project := filepath.Join(parent, "a", "b", "c")
	os.MkdirAll(project, 0755)

	os.WriteFile(filepath.Join(project, "README.md"), []byte(
		"# Deep\n\n<!-- code from=../../../secret.txt lines=1-1 -->\n<!-- /code -->\n",
	), 0644)

	cmd := exec.Command(binPath, "update")
	cmd.Dir = project
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("update should fail for ../../../ traversal")
	}
	if !strings.Contains(string(out), "escapes project directory") {
		t.Errorf("expected 'escapes project directory' error, got: %s", out)
	}
}

func TestRejectsPathUpThenIntoSibling(t *testing.T) {
	binPath := buildBinary(t)
	parent := t.TempDir()

	sibling := filepath.Join(parent, "sibling")
	os.MkdirAll(sibling, 0755)
	os.WriteFile(filepath.Join(sibling, "private.go"), []byte("package private\n"), 0644)

	project := filepath.Join(parent, "myproject")
	os.MkdirAll(project, 0755)

	os.WriteFile(filepath.Join(project, "README.md"), []byte(
		"# My Project\n\n<!-- code from=../sibling/private.go lines=1-1 -->\n<!-- /code -->\n",
	), 0644)

	cmd := exec.Command(binPath, "update")
	cmd.Dir = project
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("update should fail for ../sibling/ traversal")
	}
	if !strings.Contains(string(out), "escapes project directory") {
		t.Errorf("expected 'escapes project directory' error, got: %s", out)
	}
}

func TestRejectsPathSiblingDeepNested(t *testing.T) {
	binPath := buildBinary(t)
	parent := t.TempDir()

	deep := filepath.Join(parent, "sibling", "deep", "nested")
	os.MkdirAll(deep, 0755)
	os.WriteFile(filepath.Join(deep, "secret.go"), []byte("package secret\n"), 0644)

	project := filepath.Join(parent, "myproject")
	os.MkdirAll(project, 0755)

	os.WriteFile(filepath.Join(project, "README.md"), []byte(
		"# My Project\n\n<!-- code from=../sibling/deep/nested/secret.go lines=1-1 -->\n<!-- /code -->\n",
	), 0644)

	cmd := exec.Command(binPath, "update")
	cmd.Dir = project
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("update should fail for ../sibling/deep/nested/ traversal")
	}
	if !strings.Contains(string(out), "escapes project directory") {
		t.Errorf("expected 'escapes project directory' error, got: %s", out)
	}
}

func TestAllowsSubdirectoryFrom(t *testing.T) {
	binPath := buildBinary(t)
	dir := testdataDir("subdirectory")

	t.Cleanup(func() {
		gitCheckout(t, "test/testdata/subdirectory/")
	})

	runUpdate(t, binPath, dir)

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
