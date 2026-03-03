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
