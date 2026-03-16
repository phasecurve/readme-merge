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
		if hasher.SnippetHash(lines[i:i+lineCount]) == snippetHash {
			return i + 1, i + lineCount, true
		}
	}

	return 0, 0, false
}
