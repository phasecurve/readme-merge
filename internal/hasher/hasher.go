package hasher

import (
	"crypto/sha256"
	"fmt"
)

func ContentHash(content string) string {
	h := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", h[:8])
}
