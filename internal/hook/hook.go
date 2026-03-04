package hook

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const openMarker = "# --- readme-merge ---"
const closeMarker = "# --- /readme-merge ---"
const hookBlock = openMarker + "\nreadme-merge check --source=staged\n" + closeMarker

func Install(repoDir string) error {
	hookPath := filepath.Join(repoDir, ".git", "hooks", "pre-commit")

	existing, err := os.ReadFile(hookPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading hook: %w", err)
	}

	content := string(existing)
	if strings.Contains(content, openMarker) {
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
	startIdx := strings.Index(content, openMarker)
	endIdx := strings.Index(content, closeMarker)
	if startIdx == -1 || endIdx == -1 {
		return nil
	}

	endIdx += len(closeMarker)
	if endIdx < len(content) && content[endIdx] == '\n' {
		endIdx++
	}
	if startIdx > 0 && content[startIdx-1] == '\n' {
		startIdx--
	}

	cleaned := content[:startIdx] + content[endIdx:]
	return os.WriteFile(hookPath, []byte(cleaned), 0755)
}
