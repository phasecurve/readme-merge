package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type RenderMode string

const (
	RenderFenced RenderMode = "fenced"
	RenderRaw    RenderMode = "raw"
)

type Block struct {
	From        string
	Ref         string
	SourceStart int
	SourceEnd   int
	FileHash    string
	SnippetHash string
	Content     string
	ReadmeStart int
	ReadmeEnd   int
	IslandID    string
	IslandIndex int
	IslandTotal int
	Render      RenderMode
}

var openRe = regexp.MustCompile(
	`<!--\s*code\s+from=(\S+)` +
		`(?:\s+ref=(\S+))?` +
		`\s+lines=(\d+)-(\d+)` +
		`(?:\s+filehash=(\S+))?` +
		`(?:\s+snippethash=(\S+))?\s*-->`,
)

var closeRe = regexp.MustCompile(`<!--\s*/code\s*-->`)

var islandOpenRe = regexp.MustCompile(
	`<!--\s*island\s+file="([^"]+)"` +
		`(?:\s+repo="([^"]+)")?` +
		`(?:\s+ref="([^"]+)")?` +
		`(?:\s+filehash=(\S+))?\s*-->`,
)

var islandCloseRe = regexp.MustCompile(`<!--\s*end\s+island\s*-->`)

var linesRe = regexp.MustCompile(
	`<!--\s*lines\s+from="(\d+)"\s+to="(\d+)"` +
		`(?:\s+snippethash=(\S+))?\s*-->`,
)

func Parse(input string) ([]Block, error) {
	lines := strings.Split(input, "\n")
	var blocks []Block

	for i := 0; i < len(lines); i++ {
		if im := islandOpenRe.FindStringSubmatch(lines[i]); im != nil {
			islandBlocks, closeIdx, err := parseIsland(lines, i, im)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, islandBlocks...)
			i = closeIdx
			continue
		}

		m := openRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}

		b, closeIdx, err := parseCodeBlock(lines, i, m)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, b)
		i = closeIdx
	}

	return blocks, nil
}

func findClose(lines []string, start int, re *regexp.Regexp) int {
	for j := start + 1; j < len(lines); j++ {
		if re.MatchString(lines[j]) {
			return j
		}
	}
	return -1
}

func parseCodeBlock(lines []string, start int, m []string) (Block, int, error) {
	from := m[1]
	ref := m[2]
	startLine := m[3]
	endLine := m[4]
	fileHash := m[5]
	snippetHash := m[6]

	lineStart, err := strconv.Atoi(startLine)
	if err != nil {
		return Block{}, 0, fmt.Errorf("line %d: invalid start line %q: %w", start+1, startLine, err)
	}
	lineEnd, err := strconv.Atoi(endLine)
	if err != nil {
		return Block{}, 0, fmt.Errorf("line %d: invalid end line %q: %w", start+1, endLine, err)
	}

	closeIdx := findClose(lines, start, closeRe)
	if closeIdx == -1 {
		return Block{}, 0, fmt.Errorf("line %d: unclosed <!-- code --> block", start+1)
	}

	b := Block{
		From:        from,
		Ref:         ref,
		SourceStart: lineStart,
		SourceEnd:   lineEnd,
		FileHash:    fileHash,
		SnippetHash: snippetHash,
		ReadmeStart: start,
		ReadmeEnd:   closeIdx,
	}

	b.Content = extractFencedContent(lines[start+1 : closeIdx])
	return b, closeIdx, nil
}

func allBlank(lines []string) bool {
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			return false
		}
	}
	return true
}

func extractFencedContent(contentLines []string) string {
	if len(contentLines) == 0 || allBlank(contentLines) {
		return ""
	}

	content := strings.Join(contentLines, "\n") + "\n"
	if !strings.HasPrefix(contentLines[0], "```") {
		return content
	}

	inner := contentLines[1:]
	if len(inner) > 0 && strings.HasPrefix(inner[len(inner)-1], "```") {
		inner = inner[:len(inner)-1]
	}
	return strings.Join(inner, "\n") + "\n"
}

func parseIsland(lines []string, start int, im []string) ([]Block, int, error) {
	file := im[1]
	repo := im[2]
	ref := im[3]
	fileHash := im[4]

	from := file
	if repo != "" {
		from = repo + "//" + file
	}

	closeIdx := findClose(lines, start, islandCloseRe)
	if closeIdx == -1 {
		return nil, 0, fmt.Errorf("line %d: unclosed <!-- island --> block", start+1)
	}

	islandID := fmt.Sprintf("island-%d", start)

	type linesTag struct {
		lineIdx     int
		sourceStart int
		sourceEnd   int
		snippetHash string
	}
	var tags []linesTag

	for j := start + 1; j < closeIdx; j++ {
		lm := linesRe.FindStringSubmatch(lines[j])
		if lm == nil {
			continue
		}
		lineStart, err := strconv.Atoi(lm[1])
		if err != nil {
			return nil, 0, fmt.Errorf("line %d: invalid from %q: %w", j+1, lm[1], err)
		}
		lineEnd, err := strconv.Atoi(lm[2])
		if err != nil {
			return nil, 0, fmt.Errorf("line %d: invalid to %q: %w", j+1, lm[2], err)
		}
		tags = append(tags, linesTag{
			lineIdx:     j,
			sourceStart: lineStart,
			sourceEnd:   lineEnd,
			snippetHash: lm[3],
		})
	}

	var subBlocks []Block
	for i, tag := range tags {
		contentStart := tag.lineIdx + 1
		var contentEnd int
		if i+1 < len(tags) {
			contentEnd = tags[i+1].lineIdx
		} else {
			contentEnd = closeIdx
		}

		var content string
		if contentStart < contentEnd {
			contentLines := lines[contentStart:contentEnd]
			if !allBlank(contentLines) {
				content = strings.Join(contentLines, "\n") + "\n"
			}
		}

		subBlocks = append(subBlocks, Block{
			From:        from,
			Ref:         ref,
			SourceStart: tag.sourceStart,
			SourceEnd:   tag.sourceEnd,
			FileHash:    fileHash,
			SnippetHash: tag.snippetHash,
			Content:     content,
			ReadmeStart: start,
			ReadmeEnd:   closeIdx,
			IslandID:    islandID,
			IslandIndex: i,
			Render:      RenderRaw,
		})
	}

	if len(subBlocks) == 0 {
		return nil, 0, fmt.Errorf("line %d: island has no <lines> elements", start+1)
	}

	for i := range subBlocks {
		subBlocks[i].IslandTotal = len(subBlocks)
	}

	return subBlocks, closeIdx, nil
}

var extToLang = map[string]string{
	"go": "go", "py": "python", "js": "javascript", "ts": "typescript",
	"rs": "rust", "rb": "ruby", "java": "java", "sh": "bash",
	"bash": "bash", "zsh": "bash", "c": "c", "cpp": "cpp", "h": "c",
	"yaml": "yaml", "yml": "yaml", "json": "json", "toml": "toml",
	"sql": "sql", "html": "html", "css": "css", "md": "markdown",
}

func fenceFor(content string) string {
	maxRun := 0
	run := 0
	for _, ch := range content {
		if ch == '`' {
			run++
			if run > maxRun {
				maxRun = run
			}
		} else {
			run = 0
		}
	}
	n := 3
	if maxRun >= 3 {
		n = maxRun + 1
	}
	return strings.Repeat("`", n)
}

func langFromPath(path string) string {
	idx := strings.LastIndex(path, ".")
	if idx == -1 {
		return ""
	}
	ext := path[idx+1:]
	if l, ok := extToLang[ext]; ok {
		return l
	}
	return ext
}

func Render(original string, blocks []Block) string {
	lines := strings.Split(original, "\n")
	var result []string
	prevEnd := 0

	for idx, b := range blocks {
		if b.Render == RenderRaw && b.IslandIndex > 0 {
			continue
		}

		result = append(result, lines[prevEnd:b.ReadmeStart]...)

		if b.Render == RenderRaw {
			result = append(result, renderIsland(blocks, idx)...)
		} else {
			result = append(result, renderCodeBlock(b)...)
		}

		prevEnd = b.ReadmeEnd + 1
	}

	if prevEnd < len(lines) {
		result = append(result, lines[prevEnd:]...)
	}

	return strings.Join(result, "\n")
}

func renderCodeBlock(b Block) []string {
	var out []string

	header := fmt.Sprintf("<!-- code from=%s", b.From)
	if b.Ref != "" {
		header += " ref=" + b.Ref
	}
	header += fmt.Sprintf(" lines=%d-%d", b.SourceStart, b.SourceEnd)
	if b.FileHash != "" {
		header += " filehash=" + b.FileHash
	}
	if b.SnippetHash != "" {
		header += " snippethash=" + b.SnippetHash
	}
	header += " -->"
	out = append(out, header)

	lang := langFromPath(b.From)
	content := rewriteAnchorLinks(b.Content, b.From, b.Ref)
	content = strings.TrimRight(content, "\n")
	fence := fenceFor(content)
	out = append(out, fence+lang)
	out = append(out, content)
	out = append(out, fence)
	out = append(out, "<!-- /code -->")

	return out
}

func renderIsland(blocks []Block, startIdx int) []string {
	var out []string
	first := blocks[startIdx]

	file, repo := splitIslandFrom(first.From)

	header := fmt.Sprintf("<!-- island file=%q", file)
	if repo != "" {
		header += fmt.Sprintf(" repo=%q", repo)
	}
	if first.Ref != "" {
		header += fmt.Sprintf(" ref=%q", first.Ref)
	}
	if first.FileHash != "" {
		header += " filehash=" + first.FileHash
	}
	header += " -->"
	out = append(out, header)

	endIdx := min(startIdx+first.IslandTotal, len(blocks))
	for _, sub := range blocks[startIdx:endIdx] {
		lineTag := fmt.Sprintf("<!-- lines from=\"%d\" to=\"%d\"", sub.SourceStart, sub.SourceEnd)
		if sub.SnippetHash != "" {
			lineTag += " snippethash=" + sub.SnippetHash
		}
		lineTag += " -->"
		out = append(out, lineTag)

		content := rewriteAnchorLinks(sub.Content, sub.From, sub.Ref)
		content = strings.TrimRight(content, "\n")
		if content != "" {
			out = append(out, content)
		}
	}

	out = append(out, "<!-- end island -->")
	return out
}

func splitIslandFrom(from string) (file, repo string) {
	idx := findRepoSeparator(from)
	if idx == -1 {
		return from, ""
	}
	return from[idx+2:], from[:idx]
}

var anchorLinkRe = regexp.MustCompile(`\]\(#([^)]+)\)`)

func rewriteAnchorLinks(content, from, ref string) string {
	baseURL := blobURL(from, ref)
	if baseURL == "" {
		return content
	}
	return anchorLinkRe.ReplaceAllString(content, "]("+baseURL+"#${1})")
}

func blobURL(from, ref string) string {
	idx := findRepoSeparator(from)
	if idx == -1 {
		return ""
	}
	repoURL := from[:idx]
	filePath := from[idx+2:]

	if ref == "" {
		ref = "HEAD"
	}

	owner, repo := parseGitURL(repoURL)
	if owner == "" {
		return ""
	}

	return fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", owner, repo, ref, filePath)
}

func findRepoSeparator(from string) int {
	start := 0
	for {
		idx := strings.Index(from[start:], "//")
		if idx == -1 {
			return -1
		}
		pos := start + idx
		if pos > 0 && from[pos-1] == ':' {
			start = pos + 2
			continue
		}
		return pos
	}
}

func parseGitURL(repoURL string) (owner, repo string) {
	repoURL = strings.TrimSuffix(repoURL, ".git")

	if path, ok := strings.CutPrefix(repoURL, "git@github.com:"); ok {
		parts := strings.SplitN(path, "/", 2)
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
	}

	if _, path, ok := strings.Cut(repoURL, "github.com/"); ok {
		parts := strings.SplitN(path, "/", 2)
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
	}

	return "", ""
}
