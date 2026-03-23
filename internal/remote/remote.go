package remote

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const separator = "//"

func IsCrossRepo(from string) bool {
	return findSeparator(from) != -1
}

func ParseFromValue(from string) (repoURL, filePath string, err error) {
	idx := findSeparator(from)
	if idx == -1 {
		return "", "", fmt.Errorf("not a cross-repo reference: %s", from)
	}
	repoURL = from[:idx]
	filePath = from[idx+len(separator):]
	if strings.TrimSpace(filePath) == "" {
		return "", "", fmt.Errorf("cross-repo reference missing file path: %s", from)
	}
	return repoURL, filePath, nil
}

func CacheDir(repoURL string) string {
	cleaned := strings.TrimSuffix(repoURL, ".git")
	cleaned = strings.TrimPrefix(cleaned, "https://")
	cleaned = strings.TrimPrefix(cleaned, "http://")

	if _, path, ok := strings.Cut(cleaned, ":"); ok && !strings.Contains(cleaned[:strings.Index(cleaned, ":")], "/") {
		cleaned = path
	}

	cleaned = strings.ReplaceAll(cleaned, "/", "_")
	cleaned = strings.ReplaceAll(cleaned, ":", "_")
	return cleaned
}

type Resolver struct {
	baseDir string
	fetched map[string]bool
}

func NewResolver(cacheBaseDir string) *Resolver {
	return &Resolver{
		baseDir: cacheBaseDir,
		fetched: make(map[string]bool),
	}
}

func (r *Resolver) ReadFile(from string, ref string) (string, error) {
	repoURL, filePath, err := ParseFromValue(from)
	if err != nil {
		return "", err
	}

	if ref == "" {
		ref = "main"
	}

	bareDir := filepath.Join(r.baseDir, CacheDir(repoURL))
	cacheKey := repoURL + "@" + ref

	if err := r.ensureFetched(repoURL, ref, bareDir, cacheKey); err != nil {
		return "", err
	}

	return r.gitShow(bareDir, ref, filePath, repoURL)
}

func (r *Resolver) ensureFetched(repoURL, ref, bareDir, cacheKey string) error {
	if r.fetched[cacheKey] {
		return nil
	}

	if _, err := os.Stat(filepath.Join(bareDir, "HEAD")); os.IsNotExist(err) {
		if err := r.initBare(repoURL, bareDir); err != nil {
			return err
		}
	}

	if err := r.fetch(repoURL, ref, bareDir); err != nil {
		return err
	}

	r.fetched[cacheKey] = true
	return nil
}

func (r *Resolver) initBare(repoURL, bareDir string) error {
	if err := os.MkdirAll(bareDir, 0755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	initCmd := exec.Command("git", "init", "--bare", bareDir)
	var stderr bytes.Buffer
	initCmd.Stderr = &stderr
	if err := initCmd.Run(); err != nil {
		return fmt.Errorf("init bare repo: %s", strings.TrimSpace(stderr.String()))
	}

	addCmd := exec.Command("git", "-C", bareDir, "remote", "add", "origin", repoURL)
	addCmd.Stderr = &stderr
	if err := addCmd.Run(); err != nil {
		return fmt.Errorf("adding remote %s: %s", repoURL, strings.TrimSpace(stderr.String()))
	}

	return nil
}

type RefNotFoundError struct {
	Ref     string
	RepoURL string
	Detail  string
}

func (e *RefNotFoundError) Error() string {
	return fmt.Sprintf("ref %q not found in %s: %s", e.Ref, e.RepoURL, e.Detail)
}

func (e *RefNotFoundError) IsRefNotFound() bool { return true }

func localRef(ref string) string {
	return "refs/readme-merge/" + strings.ReplaceAll(ref, "/", "_")
}

func (r *Resolver) fetch(repoURL, ref, bareDir string) error {
	refspec := "+" + ref + ":" + localRef(ref)
	cmd := exec.Command("git", "-C", bareDir, "fetch", "--depth=1", "origin", refspec)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if strings.Contains(msg, "not found") ||
			strings.Contains(msg, "couldn't find remote ref") ||
			strings.Contains(msg, "not our ref") {
			return &RefNotFoundError{Ref: ref, RepoURL: repoURL, Detail: msg}
		}
		return fmt.Errorf("fetching %s from %s: %w (%s)", ref, repoURL, err, msg)
	}
	return nil
}

func (r *Resolver) gitShow(bareDir, ref, filePath, repoURL string) (string, error) {
	cmd := exec.Command("git", "-C", bareDir, "show", localRef(ref)+":"+filePath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("reading %s in %s ref=%s: %s", filePath, repoURL, ref, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func findSeparator(from string) int {
	start := 0
	for {
		idx := strings.Index(from[start:], separator)
		if idx == -1 {
			return -1
		}
		pos := start + idx
		if pos > 0 && from[pos-1] == ':' {
			start = pos + len(separator)
			continue
		}
		return pos
	}
}
