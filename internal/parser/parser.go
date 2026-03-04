package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func mustAtoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		panic("mustAtoi: " + err.Error())
	}
	return n
}

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

		lineStart := mustAtoi(m[2])
		lineEnd := mustAtoi(m[3])

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

func langFromPath(path string) string {
	idx := strings.LastIndex(path, ".")
	if idx == -1 {
		return ""
	}
	ext := path[idx+1:]
	langs := map[string]string{
		"go": "go", "py": "python", "js": "javascript", "ts": "typescript",
		"rs": "rust", "rb": "ruby", "java": "java", "sh": "bash",
		"bash": "bash", "zsh": "bash", "c": "c", "cpp": "cpp", "h": "c",
		"yaml": "yaml", "yml": "yaml", "json": "json", "toml": "toml",
		"sql": "sql", "html": "html", "css": "css", "md": "markdown",
	}
	if l, ok := langs[ext]; ok {
		return l
	}
	return ext
}

func Render(original string, blocks []Block) string {
	lines := strings.Split(original, "\n")
	var result []string
	prevEnd := 0

	for _, b := range blocks {
		result = append(result, lines[prevEnd:b.StartLine]...)

		header := fmt.Sprintf("<!-- code from=%s lines=%d-%d",
			b.From, b.LineStart, b.LineEnd)
		if b.FileHash != "" {
			header += " filehash=" + b.FileHash
		}
		if b.SnippetHash != "" {
			header += " snippethash=" + b.SnippetHash
		}
		header += " -->"
		result = append(result, header)

		lang := langFromPath(b.From)
		content := strings.TrimRight(b.Content, "\n")
		result = append(result, "```"+lang)
		result = append(result, content)
		result = append(result, "```")
		result = append(result, "<!-- /code -->")

		prevEnd = b.EndLine + 1
	}

	if prevEnd < len(lines) {
		result = append(result, lines[prevEnd:]...)
	}

	return strings.Join(result, "\n")
}
