package remote_test

import (
	"strings"
	"testing"

	"github.com/phasecurve/readme-merge/internal/remote"
)

const fixtureRepo = "https://github.com/phasecurve/readme-merge-examples.git"
const fixtureRef = "v1"

func TestParseFromValueSSH(t *testing.T) {
	repoURL, filePath, err := remote.ParseFromValue("git@github.com:org/repo.git//README.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repoURL != "git@github.com:org/repo.git" {
		t.Errorf("repoURL = %q, want %q", repoURL, "git@github.com:org/repo.git")
	}
	if filePath != "README.md" {
		t.Errorf("filePath = %q, want %q", filePath, "README.md")
	}
}

func TestParseFromValueHTTPS(t *testing.T) {
	repoURL, filePath, err := remote.ParseFromValue("https://github.com/org/repo.git//src/main.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repoURL != "https://github.com/org/repo.git" {
		t.Errorf("repoURL = %q, want %q", repoURL, "https://github.com/org/repo.git")
	}
	if filePath != "src/main.go" {
		t.Errorf("filePath = %q, want %q", filePath, "src/main.go")
	}
}

func TestParseFromValueMissingFilePath(t *testing.T) {
	_, _, err := remote.ParseFromValue("git@github.com:org/repo.git//")
	if err == nil {
		t.Fatal("expected error for missing file path")
	}
}

func TestCacheDirIncludesOwner(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"git@github.com:org/repo.git", "org_repo"},
		{"git@github.com:org/shared-lib.git", "org_shared-lib"},
		{"https://github.com/org/my-lib.git", "github.com_org_my-lib"},
		{"git@github.com:org/repo", "org_repo"},
	}
	for _, tt := range tests {
		got := remote.CacheDir(tt.url)
		if got != tt.want {
			t.Errorf("CacheDir(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

func TestCacheDirNoCollisionAcrossOrgs(t *testing.T) {
	dir1 := remote.CacheDir("git@github.com:org-a/utils.git")
	dir2 := remote.CacheDir("git@github.com:org-b/utils.git")
	if dir1 == dir2 {
		t.Errorf("different orgs with same repo name should produce different cache dirs, both got %q", dir1)
	}
}

func TestCacheDirDifferentForDifferentURLs(t *testing.T) {
	dir1 := remote.CacheDir("git@github.com:org/repo-a.git")
	dir2 := remote.CacheDir("git@github.com:org/repo-b.git")
	if dir1 == dir2 {
		t.Errorf("different URLs produced same cache dir: %q", dir1)
	}
}

func TestIsCrossRepo(t *testing.T) {
	if !remote.IsCrossRepo("git@github.com:org/repo.git//README.md") {
		t.Error("SSH URL should be cross-repo")
	}
	if !remote.IsCrossRepo("https://github.com/org/repo.git//README.md") {
		t.Error("HTTPS URL should be cross-repo")
	}
	if remote.IsCrossRepo("src/main.go") {
		t.Error("local path should not be cross-repo")
	}
}

func TestResolverReadFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	cacheDir := t.TempDir()
	resolver := remote.NewResolver(cacheDir)

	content, err := resolver.ReadFile(
		fixtureRepo+"//README.md",
		fixtureRef,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content == "" {
		t.Fatal("expected non-empty content")
	}
	if !strings.Contains(content, "readme-merge-examples") {
		t.Errorf("content should mention readme-merge-examples, got:\n%.200s", content)
	}
}

func TestResolverReadFileWithTagRef(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	cacheDir := t.TempDir()
	resolver := remote.NewResolver(cacheDir)

	content, err := resolver.ReadFile(
		fixtureRepo+"//src/example.go",
		fixtureRef,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(content, "func Greet(name string)") {
		t.Errorf("expected Go source content, got:\n%.200s", content)
	}
}

func TestResolverReadFileSubdirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	cacheDir := t.TempDir()
	resolver := remote.NewResolver(cacheDir)

	content, err := resolver.ReadFile(
		fixtureRepo+"//docs/style-guide.md",
		fixtureRef,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(content, "# Go Style Guide") {
		t.Errorf("expected style guide content, got:\n%.200s", content)
	}
}

func TestResolverReusesCachedClone(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	cacheDir := t.TempDir()
	resolver := remote.NewResolver(cacheDir)

	_, err := resolver.ReadFile(
		fixtureRepo+"//README.md",
		fixtureRef,
	)
	if err != nil {
		t.Fatalf("first read failed: %v", err)
	}

	_, err = resolver.ReadFile(
		fixtureRepo+"//docs/changelog.md",
		fixtureRef,
	)
	if err != nil {
		t.Fatalf("second read (cached, different file) failed: %v", err)
	}
}

func TestResolverFileNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	cacheDir := t.TempDir()
	resolver := remote.NewResolver(cacheDir)

	_, err := resolver.ReadFile(
		fixtureRepo+"//nonexistent-file.txt",
		fixtureRef,
	)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestParseFromValueNoSeparator(t *testing.T) {
	_, _, err := remote.ParseFromValue("src/main.go")
	if err == nil {
		t.Fatal("expected error for local path")
	}
}
