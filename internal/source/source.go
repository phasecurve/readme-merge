package source

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
