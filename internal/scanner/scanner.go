package scanner

import (
	"strings"

	"github.com/phasecurve/readme-merge/internal/hasher"
)

func FindSnippet(fileContent string, snippetHash string, lineCount int) (startLine, endLine int, found bool) {
	trimmed := strings.TrimRight(fileContent, "\n")
	lineOffsets := buildLineOffsets(trimmed)

	if lineCount <= 0 || lineCount > len(lineOffsets) {
		return 0, 0, false
	}

	for i := 0; i <= len(lineOffsets)-lineCount; i++ {
		candidate := extractWindow(trimmed, lineOffsets, i, lineCount)
		if hasher.ContentHash(candidate) == snippetHash {
			return i + 1, i + lineCount, true
		}
	}

	return 0, 0, false
}

func buildLineOffsets(content string) []int {
	offsets := []int{0}
	for i := range len(content) {
		if content[i] == '\n' {
			offsets = append(offsets, i+1)
		}
	}
	return offsets
}

func extractWindow(content string, offsets []int, start, count int) string {
	begin := offsets[start]
	var end int
	if start+count < len(offsets) {
		end = offsets[start+count] - 1
	} else {
		end = len(content)
	}
	return content[begin:end] + "\n"
}
