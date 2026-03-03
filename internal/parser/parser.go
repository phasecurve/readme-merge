package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Block struct {
	From        string
	LineStart   int
	LineEnd     int
	FileHash    string
	SnippetHash string
	Content     string
	StartLine   int
	EndLine     int
}

var openRe = regexp.MustCompile(
	`<!--\s*code\s+from=(\S+)\s+lines=(\d+)-(\d+)` +
		`(?:\s+filehash=(\S+))?` +
		`(?:\s+snippethash=(\S+))?\s*-->`,
)

var closeRe = regexp.MustCompile(`<!--\s*/code\s*-->`)

func Parse(input string) ([]Block, error) {
	lines := strings.Split(input, "\n")
	var blocks []Block

	for i := 0; i < len(lines); i++ {
		m := openRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}

		lineStart, _ := strconv.Atoi(m[2])
		lineEnd, _ := strconv.Atoi(m[3])

		b := Block{
			From:        m[1],
			LineStart:   lineStart,
			LineEnd:     lineEnd,
			FileHash:    m[4],
			SnippetHash: m[5],
			StartLine:   i,
		}

		closeIdx := -1
		for j := i + 1; j < len(lines); j++ {
			if closeRe.MatchString(lines[j]) {
				closeIdx = j
				break
			}
		}
		if closeIdx == -1 {
			return nil, fmt.Errorf("line %d: unclosed <!-- code --> block", i+1)
		}

		b.EndLine = closeIdx

		contentLines := lines[i+1 : closeIdx]
		if len(contentLines) > 0 {
			content := strings.Join(contentLines, "\n") + "\n"
			if strings.HasPrefix(contentLines[0], "```") {
				inner := contentLines[1:]
				if len(inner) > 0 && strings.HasPrefix(inner[len(inner)-1], "```") {
					inner = inner[:len(inner)-1]
				}
				content = strings.Join(inner, "\n") + "\n"
			}
			if strings.TrimSpace(strings.Join(contentLines, "")) != "" {
				b.Content = content
			}
		}

		blocks = append(blocks, b)
		i = closeIdx
	}

	return blocks, nil
}
