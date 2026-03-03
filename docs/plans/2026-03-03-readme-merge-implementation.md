# readme-merge Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Go CLI that keeps README code examples in sync with real source code via placeholder comments, two-hash staleness detection, and self-healing line references.

**Architecture:** Bottom-up build: core libraries (parser, extractor, hasher, scanner) first, then the source resolver, then the engine that orchestrates them, then CLI commands on top. Each layer is tested before the next is built.

**Tech Stack:** Go 1.26, standard library only (no cobra - use `flag` + subcommands). SHA-256 for hashing. `os/exec` for git commands in source resolver.

---

### Task 1: Project scaffolding

**Files:**
- Create: `go.mod`
- Create: `cmd/readme-merge/main.go`

**Step 1: Initialise Go module**

Run: `cd ~/dev/readme-merge && go mod init github.com/phasecurve/readme-merge`
Expected: `go.mod` created

**Step 2: Write minimal main.go**

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: readme-merge <update|check|hook>")
		os.Exit(1)
	}
	fmt.Println("readme-merge:", os.Args[1])
}
```

**Step 3: Verify it builds and runs**

Run: `go run ./cmd/readme-merge update`
Expected: `readme-merge: update`

**Step 4: Commit**

```bash
git add go.mod cmd/
git commit -m "chore: scaffold project with go module and main entry point"
```

---

### Task 2: Hasher - SHA-256 content hashing

**Files:**
- Create: `internal/hasher/hasher.go`
- Create: `internal/hasher/hasher_test.go`

**Step 1: Write the failing test**

```go
package hasher_test

import (
	"testing"

	"github.com/phasecurve/readme-merge/internal/hasher"
)

func TestContentHash(t *testing.T) {
	input := "func main() {\n\tfmt.Println(\"hello\")\n}\n"
	got := hasher.ContentHash(input)

	if len(got) != 16 {
		t.Errorf("expected 16 hex chars, got %d: %q", len(got), got)
	}

	got2 := hasher.ContentHash(input)
	if got != got2 {
		t.Errorf("same input produced different hashes: %q vs %q", got, got2)
	}

	different := hasher.ContentHash("something else")
	if got == different {
		t.Errorf("different inputs produced same hash")
	}
}

func TestContentHashEmpty(t *testing.T) {
	got := hasher.ContentHash("")
	if len(got) != 16 {
		t.Errorf("expected 16 hex chars for empty string, got %d", len(got))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd ~/dev/readme-merge && go test ./internal/hasher/ -v`
Expected: FAIL - package doesn't exist

**Step 3: Write minimal implementation**

```go
package hasher

import (
	"crypto/sha256"
	"fmt"
)

func ContentHash(content string) string {
	h := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", h[:8])
}
```

**Step 4: Run tests to verify they pass**

Run: `cd ~/dev/readme-merge && go test ./internal/hasher/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/hasher/
git commit -m "feat(hasher): add SHA-256 content hashing truncated to 16 hex chars"
```

---

### Task 3: Parser - parse README placeholder comments

**Files:**
- Create: `internal/parser/parser.go`
- Create: `internal/parser/parser_test.go`

The parser finds `<!-- code from=... lines=... -->` blocks in markdown and returns
structured data. It also rewrites blocks with updated content/hashes.

**Step 1: Write the failing test for parsing placeholders**

```go
package parser_test

import (
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
	if b.LineStart != 10 || b.LineEnd != 25 {
		t.Errorf("Lines = %d-%d, want 10-25", b.LineStart, b.LineEnd)
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
```

**Step 2: Run test to verify it fails**

Run: `cd ~/dev/readme-merge && go test ./internal/parser/ -v`
Expected: FAIL - package doesn't exist

**Step 3: Write the Block type and Parse function**

```go
package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Block struct {
	From        string
	LineStart   int
	LineEnd     int
	FileHash    string
	SnippetHash string
	Content     string
	StartLine   int
	EndLine     int
}

var openRe = regexp.MustCompile(
	`<!--\s*code\s+from=(\S+)\s+lines=(\d+)-(\d+)` +
		`(?:\s+filehash=(\S+))?` +
		`(?:\s+snippethash=(\S+))?\s*-->`,
)

var closeRe = regexp.MustCompile(`<!--\s*/code\s*-->`)

func Parse(input string) ([]Block, error) {
	lines := strings.Split(input, "\n")
	var blocks []Block

	for i := 0; i < len(lines); i++ {
		m := openRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}

		lineStart, _ := strconv.Atoi(m[2])
		lineEnd, _ := strconv.Atoi(m[3])

		b := Block{
			From:        m[1],
			LineStart:   lineStart,
			LineEnd:     lineEnd,
			FileHash:    m[4],
			SnippetHash: m[5],
			StartLine:   i,
		}

		closeIdx := -1
		for j := i + 1; j < len(lines); j++ {
			if closeRe.MatchString(lines[j]) {
				closeIdx = j
				break
			}
		}
		if closeIdx == -1 {
			return nil, fmt.Errorf("line %d: unclosed <!-- code --> block", i+1)
		}

		b.EndLine = closeIdx

		contentLines := lines[i+1 : closeIdx]
		if len(contentLines) > 0 {
			content := strings.Join(contentLines, "\n") + "\n"
			if strings.HasPrefix(contentLines[0], "```") {
				inner := contentLines[1:]
				if len(inner) > 0 && strings.HasPrefix(inner[len(inner)-1], "```") {
					inner = inner[:len(inner)-1]
				}
				content = strings.Join(inner, "\n") + "\n"
			}
			if strings.TrimSpace(strings.Join(contentLines, "")) != "" {
				b.Content = content
			}
		}

		blocks = append(blocks, b)
		i = closeIdx
	}

	return blocks, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `cd ~/dev/readme-merge && go test ./internal/parser/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/parser/
git commit -m "feat(parser): parse README placeholder comments into Block structs"
```

---

### Task 4: Parser - render blocks back into README

**Files:**
- Modify: `internal/parser/parser.go`
- Modify: `internal/parser/parser_test.go`

**Step 1: Write the failing test for rendering**

```go
func TestRenderNewBlock(t *testing.T) {
	original := "# Title\n\n<!-- code from=examples/client.go lines=10-12 -->\n<!-- /code -->\n\nMore text.\n"

	blocks, _ := parser.Parse(original)
	blocks[0].Content = "fmt.Println(\"hello\")\n"
	blocks[0].FileHash = "aaaa1234aaaa1234"
	blocks[0].SnippetHash = "bbbb1234bbbb1234"
	blocks[0].LineStart = 10
	blocks[0].LineEnd = 12

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
```

**Step 2: Run test to verify it fails**

Run: `cd ~/dev/readme-merge && go test ./internal/parser/ -v`
Expected: FAIL - Render not defined

**Step 3: Implement Render function**

The Render function takes the original README text and updated blocks, and
produces the new README. It infers the code fence language from the file
extension.

```go
func langFromPath(path string) string {
	idx := strings.LastIndex(path, ".")
	if idx == -1 {
		return ""
	}
	ext := path[idx+1:]
	langs := map[string]string{
		"go": "go", "py": "python", "js": "javascript", "ts": "typescript",
		"rs": "rust", "rb": "ruby", "java": "java", "sh": "bash",
		"bash": "bash", "zsh": "bash", "c": "c", "cpp": "cpp", "h": "c",
		"yaml": "yaml", "yml": "yaml", "json": "json", "toml": "toml",
		"sql": "sql", "html": "html", "css": "css", "md": "markdown",
	}
	if l, ok := langs[ext]; ok {
		return l
	}
	return ext
}

func Render(original string, blocks []Block) string {
	lines := strings.Split(original, "\n")
	var result []string
	prevEnd := 0

	for _, b := range blocks {
		result = append(result, lines[prevEnd:b.StartLine]...)

		header := fmt.Sprintf("<!-- code from=%s lines=%d-%d",
			b.From, b.LineStart, b.LineEnd)
		if b.FileHash != "" {
			header += " filehash=" + b.FileHash
		}
		if b.SnippetHash != "" {
			header += " snippethash=" + b.SnippetHash
		}
		header += " -->"
		result = append(result, header)

		lang := langFromPath(b.From)
		content := strings.TrimRight(b.Content, "\n")
		result = append(result, "```"+lang)
		result = append(result, content)
		result = append(result, "```")
		result = append(result, "<!-- /code -->")

		prevEnd = b.EndLine + 1
	}

	if prevEnd < len(lines) {
		result = append(result, lines[prevEnd:]...)
	}

	return strings.Join(result, "\n")
}
```

**Step 4: Run tests to verify they pass**

Run: `cd ~/dev/readme-merge && go test ./internal/parser/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/parser/
git commit -m "feat(parser): render updated blocks back into README markdown"
```

---

### Task 5: Extractor - read source files and extract line ranges

**Files:**
- Create: `internal/extractor/extractor.go`
- Create: `internal/extractor/extractor_test.go`

**Step 1: Write the failing test**

```go
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
```

**Step 2: Run test to verify it fails**

Run: `cd ~/dev/readme-merge && go test ./internal/extractor/ -v`
Expected: FAIL

**Step 3: Write minimal implementation**

```go
package extractor

import (
	"fmt"
	"os"
	"strings"
)

func Extract(path string, lineStart, lineEnd int) (snippet string, fileContent string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("reading %s: %w", path, err)
	}

	fileContent = string(data)
	lines := strings.Split(fileContent, "\n")

	if lineEnd > len(lines) || lineStart < 1 {
		return "", fileContent, fmt.Errorf(
			"%s: line range %d-%d out of bounds (file has %d lines)",
			path, lineStart, lineEnd, len(lines),
		)
	}

	selected := lines[lineStart-1 : lineEnd]
	snippet = strings.Join(selected, "\n") + "\n"
	return snippet, fileContent, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `cd ~/dev/readme-merge && go test ./internal/extractor/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/extractor/
git commit -m "feat(extractor): extract line ranges from source files"
```

---

### Task 6: Scanner - self-healing snippet relocation

**Files:**
- Create: `internal/scanner/scanner.go`
- Create: `internal/scanner/scanner_test.go`

The scanner searches a file for a contiguous block of lines whose hash matches
a known snippet hash. Returns the new line range if found.

**Step 1: Write the failing test**

```go
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
```

**Step 2: Run test to verify it fails**

Run: `cd ~/dev/readme-merge && go test ./internal/scanner/ -v`
Expected: FAIL

**Step 3: Write the FindSnippet function**

```go
package scanner

import (
	"strings"

	"github.com/phasecurve/readme-merge/internal/hasher"
)

func FindSnippet(fileContent string, snippetHash string, lineCount int) (startLine, endLine int, found bool) {
	lines := strings.Split(strings.TrimRight(fileContent, "\n"), "\n")

	if lineCount <= 0 || lineCount > len(lines) {
		return 0, 0, false
	}

	for i := 0; i <= len(lines)-lineCount; i++ {
		candidate := strings.Join(lines[i:i+lineCount], "\n") + "\n"
		if hasher.ContentHash(candidate) == snippetHash {
			return i + 1, i + lineCount, true
		}
	}

	return 0, 0, false
}
```

**Step 4: Run tests to verify they pass**

Run: `cd ~/dev/readme-merge && go test ./internal/scanner/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/scanner/
git commit -m "feat(scanner): self-healing snippet relocation via content hash scan"
```

---

### Task 7: Source resolver - worktree and git ref file reading

**Files:**
- Create: `internal/source/source.go`
- Create: `internal/source/source_test.go`

**Step 1: Write the failing test for worktree (file-based) reading**

```go
package source_test

import (
	"os"
	"path/filepath"
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
```

**Step 2: Run test to verify it fails**

Run: `cd ~/dev/readme-merge && go test ./internal/source/ -v`
Expected: FAIL

**Step 3: Write the Resolver**

```go
package source

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Resolver struct {
	ref     string
	baseDir string
}

func NewResolver(ref string, baseDir string) *Resolver {
	return &Resolver{ref: ref, baseDir: baseDir}
}

func (r *Resolver) ReadFile(path string) (string, error) {
	if r.ref == "" {
		return r.readWorktree(path)
	}
	return r.readGitRef(path)
}

func (r *Resolver) readWorktree(path string) (string, error) {
	full := filepath.Join(r.baseDir, path)
	data, err := os.ReadFile(full)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", full, err)
	}
	return string(data), nil
}

func (r *Resolver) readGitRef(path string) (string, error) {
	var ref string
	if r.ref == "staged" {
		ref = ":" + path
	} else {
		ref = r.ref + ":" + path
	}

	cmd := exec.Command("git", "-C", r.baseDir, "show", ref)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git show %s: %w", ref, err)
	}
	return string(out), nil
}

func (r *Resolver) IsGitRequired() bool {
	return r.ref != ""
}

func IsGitRepo(dir string) bool {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

func ValidateSource(ref string, baseDir string) error {
	if ref == "" {
		return nil
	}
	if !IsGitRepo(baseDir) {
		return fmt.Errorf("--source=%s requires a git repository", ref)
	}
	return nil
}
```

Note: `strings` import may be unused - remove if so at test time.

**Step 4: Run tests to verify they pass**

Run: `cd ~/dev/readme-merge && go test ./internal/source/ -v`
Expected: PASS

**Step 5: Write additional test for git ref reading (requires git repo)**

```go
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
```

**Step 6: Run all source tests**

Run: `cd ~/dev/readme-merge && go test ./internal/source/ -v`
Expected: PASS

**Step 7: Commit**

```bash
git add internal/source/
git commit -m "feat(source): file resolver with worktree and git ref support"
```

---

### Task 8: Engine - orchestrate check and update logic

**Files:**
- Create: `internal/engine/engine.go`
- Create: `internal/engine/engine_test.go`

This is the core orchestrator. It takes a README path, a source resolver, and
runs the two-hash detection logic across all blocks.

**Step 1: Write the failing test for update (new placeholder)**

```go
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
```

**Step 2: Run test to verify it fails**

Run: `cd ~/dev/readme-merge && go test ./internal/engine/ -v`
Expected: FAIL

**Step 3: Write the engine**

```go
package engine

import (
	"fmt"
	"os"
	"strings"

	"github.com/phasecurve/readme-merge/internal/extractor"
	"github.com/phasecurve/readme-merge/internal/hasher"
	"github.com/phasecurve/readme-merge/internal/parser"
	"github.com/phasecurve/readme-merge/internal/scanner"
	"github.com/phasecurve/readme-merge/internal/source"
)

type UpdateResult struct {
	Output  string
	Updated int
	Healed  int
}

type CheckResult struct {
	Stale    []StaleBlock
	Unhashed []parser.Block
	Healed   int
	Fresh    int
}

type StaleBlock struct {
	Block   parser.Block
	Message string
}

func Update(readmePath string, resolver *source.Resolver) (*UpdateResult, error) {
	data, err := os.ReadFile(readmePath)
	if err != nil {
		return nil, fmt.Errorf("reading README: %w", err)
	}

	content := string(data)
	blocks, err := parser.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("parsing README: %w", err)
	}

	result := &UpdateResult{}

	for i := range blocks {
		b := &blocks[i]

		fileContent, err := resolver.ReadFile(b.From)
		if err != nil {
			return nil, fmt.Errorf("block %s: %w", b.From, err)
		}

		lines := strings.Split(fileContent, "\n")
		if b.LineEnd > len(lines) || b.LineStart < 1 {
			return nil, fmt.Errorf("block %s: line range %d-%d out of bounds (%d lines)",
				b.From, b.LineStart, b.LineEnd, len(lines))
		}

		selected := lines[b.LineStart-1 : b.LineEnd]
		snippet := strings.Join(selected, "\n") + "\n"

		b.Content = snippet
		b.FileHash = hasher.ContentHash(fileContent)
		b.SnippetHash = hasher.ContentHash(snippet)
		result.Updated++
	}

	result.Output = parser.Render(content, blocks)
	os.WriteFile(readmePath, []byte(result.Output), 0644)
	return result, nil
}

func Check(readmePath string, resolver *source.Resolver) (*CheckResult, error) {
	data, err := os.ReadFile(readmePath)
	if err != nil {
		return nil, fmt.Errorf("reading README: %w", err)
	}

	content := string(data)
	blocks, err := parser.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("parsing README: %w", err)
	}

	result := &CheckResult{}
	needsWrite := false

	for i := range blocks {
		b := &blocks[i]

		if b.FileHash == "" || b.SnippetHash == "" {
			result.Unhashed = append(result.Unhashed, *b)
			continue
		}

		fileContent, err := resolver.ReadFile(b.From)
		if err != nil {
			return nil, fmt.Errorf("block %s: %w", b.From, err)
		}

		currentFileHash := hasher.ContentHash(fileContent)
		if currentFileHash == b.FileHash {
			result.Fresh++
			continue
		}

		lines := strings.Split(fileContent, "\n")
		lineCount := b.LineEnd - b.LineStart + 1

		if b.LineEnd <= len(lines) {
			selected := lines[b.LineStart-1 : b.LineEnd]
			candidate := strings.Join(selected, "\n") + "\n"
			if hasher.ContentHash(candidate) == b.SnippetHash {
				b.FileHash = currentFileHash
				result.Fresh++
				needsWrite = true
				continue
			}
		}

		start, end, found := scanner.FindSnippet(fileContent, b.SnippetHash, lineCount)
		if found {
			b.LineStart = start
			b.LineEnd = end
			b.FileHash = currentFileHash
			result.Healed++
			needsWrite = true
			continue
		}

		result.Stale = append(result.Stale, StaleBlock{
			Block:   *b,
			Message: fmt.Sprintf("%s lines %d-%d: content changed", b.From, b.LineStart, b.LineEnd),
		})
	}

	if needsWrite {
		output := parser.Render(content, blocks)
		os.WriteFile(readmePath, []byte(output), 0644)
	}

	return result, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `cd ~/dev/readme-merge && go test ./internal/engine/ -v`
Expected: PASS

**Step 5: Write test for self-healing**

```go
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
```

**Step 6: Run all engine tests**

Run: `cd ~/dev/readme-merge && go test ./internal/engine/ -v`
Expected: PASS

**Step 7: Commit**

```bash
git add internal/engine/
git commit -m "feat(engine): orchestrate check and update with two-hash detection and self-healing"
```

---

### Task 9: CLI - wire up update and check commands

**Files:**
- Modify: `cmd/readme-merge/main.go`

**Step 1: Write the CLI with subcommands**

```go
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/phasecurve/readme-merge/internal/engine"
	"github.com/phasecurve/readme-merge/internal/source"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "update":
		runUpdate(os.Args[2:])
	case "check":
		runCheck(os.Args[2:])
	case "hook":
		runHook(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: readme-merge <update|check|hook> [flags]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "commands:")
	fmt.Fprintln(os.Stderr, "  update    populate/refresh all code placeholders")
	fmt.Fprintln(os.Stderr, "  check     verify all placeholders are fresh (exit 1 if stale)")
	fmt.Fprintln(os.Stderr, "  hook      install/uninstall git pre-commit hook")
}

func findReadme(dir string) (string, error) {
	candidates := []string{"README.md", "readme.md", "Readme.md"}
	for _, c := range candidates {
		path := filepath.Join(dir, c)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no README.md found in %s", dir)
}

func runUpdate(args []string) {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	sourceRef := fs.String("source", "", "source ref: staged, HEAD, or git ref (default: worktree)")
	readme := fs.String("file", "", "path to README (default: auto-detect)")
	fs.Parse(args)

	dir, _ := os.Getwd()

	if err := source.ValidateSource(*sourceRef, dir); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	readmePath := *readme
	if readmePath == "" {
		var err error
		readmePath, err = findReadme(dir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	}

	resolver := source.NewResolver(*sourceRef, dir)
	result, err := engine.Update(readmePath, resolver)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	fmt.Printf("updated %d placeholder(s)\n", result.Updated)
}

func runCheck(args []string) {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	sourceRef := fs.String("source", "", "source ref: staged, HEAD, or git ref (default: worktree)")
	readme := fs.String("file", "", "path to README (default: auto-detect)")
	fs.Parse(args)

	dir, _ := os.Getwd()

	if err := source.ValidateSource(*sourceRef, dir); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	readmePath := *readme
	if readmePath == "" {
		var err error
		readmePath, err = findReadme(dir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	}

	resolver := source.NewResolver(*sourceRef, dir)
	result, err := engine.Check(readmePath, resolver)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	exitCode := 0

	if len(result.Unhashed) > 0 {
		fmt.Fprintf(os.Stderr, "%d unhashed placeholder(s) - run 'readme-merge update' first:\n", len(result.Unhashed))
		for _, b := range result.Unhashed {
			fmt.Fprintf(os.Stderr, "  %s lines %d-%d\n", b.From, b.LineStart, b.LineEnd)
		}
		exitCode = 1
	}

	if len(result.Stale) > 0 {
		fmt.Fprintf(os.Stderr, "%d stale placeholder(s):\n", len(result.Stale))
		for _, s := range result.Stale {
			fmt.Fprintf(os.Stderr, "  %s\n", s.Message)
		}
		exitCode = 1
	}

	if result.Healed > 0 {
		fmt.Printf("self-healed %d placeholder(s) (lines shifted)\n", result.Healed)
	}

	fmt.Printf("%d placeholder(s) fresh\n", result.Fresh)

	os.Exit(exitCode)
}

func runHook(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: readme-merge hook <install|uninstall>")
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "hook command not yet implemented")
	os.Exit(1)
}
```

**Step 2: Build and test manually**

Run: `cd ~/dev/readme-merge && go build ./cmd/readme-merge && echo "build ok"`
Expected: `build ok`

**Step 3: Commit**

```bash
git add cmd/readme-merge/main.go
git commit -m "feat(cli): wire up update and check commands with --source and --file flags"
```

---

### Task 10: Hook - install/uninstall pre-commit hook

**Files:**
- Create: `internal/hook/hook.go`
- Create: `internal/hook/hook_test.go`
- Modify: `cmd/readme-merge/main.go` (wire up runHook)

**Step 1: Write the failing test**

```go
package hook_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phasecurve/readme-merge/internal/hook"
)

func TestInstallNewHook(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, ".git", "hooks")
	os.MkdirAll(hooksDir, 0755)

	err := hook.Install(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(hooksDir, "pre-commit"))
	if err != nil {
		t.Fatalf("hook file not created: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "readme-merge check --source=staged") {
		t.Errorf("hook missing readme-merge command:\n%s", content)
	}
}

func TestInstallExistingHook(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, ".git", "hooks")
	os.MkdirAll(hooksDir, 0755)
	os.WriteFile(filepath.Join(hooksDir, "pre-commit"), []byte("#!/bin/sh\necho existing\n"), 0755)

	hook.Install(dir)

	data, _ := os.ReadFile(filepath.Join(hooksDir, "pre-commit"))
	content := string(data)
	if !strings.Contains(content, "echo existing") {
		t.Errorf("existing hook content lost")
	}
	if !strings.Contains(content, "readme-merge") {
		t.Errorf("readme-merge not appended")
	}
}

func TestUninstall(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, ".git", "hooks")
	os.MkdirAll(hooksDir, 0755)

	hook.Install(dir)
	hook.Uninstall(dir)

	data, _ := os.ReadFile(filepath.Join(hooksDir, "pre-commit"))
	if strings.Contains(string(data), "readme-merge") {
		t.Errorf("readme-merge not removed from hook")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd ~/dev/readme-merge && go test ./internal/hook/ -v`
Expected: FAIL

**Step 3: Write the hook install/uninstall**

```go
package hook

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const marker = "# --- readme-merge ---"
const hookBlock = `# --- readme-merge ---
readme-merge check --source=staged
# --- /readme-merge ---`

func Install(repoDir string) error {
	hookPath := filepath.Join(repoDir, ".git", "hooks", "pre-commit")

	existing, err := os.ReadFile(hookPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading hook: %w", err)
	}

	content := string(existing)
	if strings.Contains(content, marker) {
		return nil
	}

	if len(content) == 0 {
		content = "#!/bin/sh\n"
	}

	content = content + "\n" + hookBlock + "\n"

	return os.WriteFile(hookPath, []byte(content), 0755)
}

func Uninstall(repoDir string) error {
	hookPath := filepath.Join(repoDir, ".git", "hooks", "pre-commit")

	data, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading hook: %w", err)
	}

	content := string(data)
	startIdx := strings.Index(content, "# --- readme-merge ---")
	endIdx := strings.Index(content, "# --- /readme-merge ---")
	if startIdx == -1 || endIdx == -1 {
		return nil
	}

	endIdx += len("# --- /readme-merge ---")
	if endIdx < len(content) && content[endIdx] == '\n' {
		endIdx++
	}
	if startIdx > 0 && content[startIdx-1] == '\n' {
		startIdx--
	}

	cleaned := content[:startIdx] + content[endIdx:]
	return os.WriteFile(hookPath, []byte(cleaned), 0755)
}
```

**Step 4: Run tests to verify they pass**

Run: `cd ~/dev/readme-merge && go test ./internal/hook/ -v`
Expected: PASS

**Step 5: Wire into CLI - update runHook in main.go**

Replace the placeholder `runHook` function:

```go
func runHook(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: readme-merge hook <install|uninstall>")
		os.Exit(1)
	}

	dir, _ := os.Getwd()

	switch args[0] {
	case "install":
		if err := hook.Install(dir); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		fmt.Println("pre-commit hook installed")
	case "uninstall":
		if err := hook.Uninstall(dir); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		fmt.Println("pre-commit hook removed")
	default:
		fmt.Fprintln(os.Stderr, "usage: readme-merge hook <install|uninstall>")
		os.Exit(1)
	}
}
```

Add `hook` import to main.go.

**Step 6: Build and verify**

Run: `cd ~/dev/readme-merge && go build ./cmd/readme-merge && echo "build ok"`
Expected: `build ok`

**Step 7: Commit**

```bash
git add internal/hook/ cmd/readme-merge/main.go
git commit -m "feat(hook): install/uninstall git pre-commit hook"
```

---

### Task 11: End-to-end integration test

**Files:**
- Create: `test/integration_test.go`

**Step 1: Write an end-to-end test**

```go
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

	exec.Command(binPath, "update").Run()

	os.WriteFile(filepath.Join(dir, "example.go"), []byte(
		"new first line\nline1\ntarget code\nline3\n",
	), 0644)

	cmd := exec.Command(binPath, "check")
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
```

**Step 2: Run the integration tests**

Run: `cd ~/dev/readme-merge && go test ./test/ -v`
Expected: PASS

**Step 3: Run all tests**

Run: `cd ~/dev/readme-merge && go test ./... -v`
Expected: ALL PASS

**Step 4: Commit**

```bash
git add test/
git commit -m "test: add end-to-end integration tests"
```

---

### Task 12: Final build and install

**Step 1: Build the binary**

Run: `cd ~/dev/readme-merge && go build -o readme-merge ./cmd/readme-merge && echo "build ok"`
Expected: `build ok`

**Step 2: Run full test suite one final time**

Run: `cd ~/dev/readme-merge && go test ./... -v -count=1`
Expected: ALL PASS

**Step 3: Commit any remaining changes**

```bash
git add -A
git commit -m "chore: final build verification"
```
