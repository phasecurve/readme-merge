package source

import (
	"bytes"
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
	if err := validatePath(r.baseDir, path); err != nil {
		return "", err
	}
	if r.ref == "" {
		return r.readWorktree(path)
	}
	return r.readGitRef(path)
}

func validatePath(baseDir, path string) error {
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute paths not allowed: %s", path)
	}
	cleaned := filepath.Clean(path)
	if strings.HasPrefix(cleaned, "..") {
		return fmt.Errorf("path escapes project directory: %s", path)
	}
	full := filepath.Join(baseDir, cleaned)
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return fmt.Errorf("resolving base directory: %w", err)
	}
	absFull, err := filepath.Abs(full)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}
	if !strings.HasPrefix(absFull, absBase+string(filepath.Separator)) && absFull != absBase {
		return fmt.Errorf("path escapes project directory: %s", path)
	}
	return nil
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
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return "", fmt.Errorf("git show %s: %s", ref, msg)
		}
		return "", fmt.Errorf("git show %s: %w", ref, err)
	}
	return string(out), nil
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
