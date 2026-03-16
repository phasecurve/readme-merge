package hasher

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

func ContentHash(content string) string {
	h := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", h[:8])
}

func SnippetHash(lines []string) string {
	return ContentHash(strings.Join(lines, "\n") + "\n")
}
