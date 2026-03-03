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
