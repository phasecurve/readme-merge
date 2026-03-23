package engine

import (
	"fmt"
	"strings"

	"github.com/phasecurve/readme-merge/internal/hasher"
	"github.com/phasecurve/readme-merge/internal/parser"
	"github.com/phasecurve/readme-merge/internal/scanner"
)

type FileReader interface {
	ReadFile(path string, ref string) (string, error)
}

type refNotFound interface {
	IsRefNotFound() bool
}

type UpdateResult struct {
	Output      string
	Updated     int
	Healed      int
	Unreachable []UnreachableBlock
}

type CheckResult struct {
	Output      string
	Stale       []StaleBlock
	Unhashed    []parser.Block
	FreshBlocks []parser.Block
	Healed      int
	Fresh       int
	Unreachable []UnreachableBlock
}

type StaleBlock struct {
	Block   parser.Block
	Message string
}

type UnreachableBlock struct {
	Block   parser.Block
	Message string
}

type fileCache struct {
	reader FileReader
	cache  map[string]string
}

func newFileCache(reader FileReader) *fileCache {
	return &fileCache{reader: reader, cache: map[string]string{}}
}

type readResult struct {
	content     string
	unreachable bool
	message     string
}

func (fc *fileCache) read(from, ref string) (readResult, error) {
	key := from + "\x00" + ref
	if content, ok := fc.cache[key]; ok {
		return readResult{content: content}, nil
	}

	content, err := fc.reader.ReadFile(from, ref)
	if err != nil {
		if rnf, ok := err.(refNotFound); ok && rnf.IsRefNotFound() {
			return readResult{unreachable: true, message: err.Error()}, nil
		}
		return readResult{}, fmt.Errorf("block %s: %w", from, err)
	}

	fc.cache[key] = content
	return readResult{content: content}, nil
}

func Update(content string, reader FileReader) (*UpdateResult, error) {
	blocks, err := parser.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("parsing README: %w", err)
	}

	result := &UpdateResult{}
	fc := newFileCache(reader)

	for i := range blocks {
		b := &blocks[i]

		r, err := fc.read(b.From, b.Ref)
		if err != nil {
			return nil, err
		}
		if r.unreachable {
			result.Unreachable = append(result.Unreachable, UnreachableBlock{Block: *b, Message: r.message})
			continue
		}

		fileContent := r.content
		lines := strings.Split(fileContent, "\n")

		if b.SnippetHash != "" {
			lineCount := b.SourceEnd - b.SourceStart + 1

			if b.SourceEnd <= len(lines) {
				selected := lines[b.SourceStart-1 : b.SourceEnd]
				if hasher.SnippetHash(selected) == b.SnippetHash {
					b.Content = strings.Join(selected, "\n") + "\n"
					b.FileHash = hasher.ContentHash(fileContent)
					result.Updated++
					continue
				}
			}

			start, end, found := scanner.FindSnippet(fileContent, b.SnippetHash, lineCount)
			if found {
				selected := lines[start-1 : end]
				b.SourceStart = start
				b.SourceEnd = end
				b.Content = strings.Join(selected, "\n") + "\n"
				b.FileHash = hasher.ContentHash(fileContent)
				result.Healed++
				result.Updated++
				continue
			}
		}

		if b.SourceEnd > len(lines) || b.SourceStart < 1 {
			return nil, fmt.Errorf("block %s: line range %d-%d out of bounds (%d lines)",
				b.From, b.SourceStart, b.SourceEnd, len(lines))
		}

		selected := lines[b.SourceStart-1 : b.SourceEnd]

		b.Content = strings.Join(selected, "\n") + "\n"
		b.FileHash = hasher.ContentHash(fileContent)
		b.SnippetHash = hasher.SnippetHash(selected)
		result.Updated++
	}

	result.Output = parser.Render(content, blocks)
	return result, nil
}

func Check(content string, reader FileReader) (*CheckResult, error) {
	blocks, err := parser.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("parsing README: %w", err)
	}

	result := &CheckResult{}
	needsRender := false
	fc := newFileCache(reader)

	for i := range blocks {
		b := &blocks[i]

		if b.FileHash == "" || b.SnippetHash == "" {
			result.Unhashed = append(result.Unhashed, *b)
			continue
		}

		r, err := fc.read(b.From, b.Ref)
		if err != nil {
			return nil, err
		}
		if r.unreachable {
			result.Unreachable = append(result.Unreachable, UnreachableBlock{Block: *b, Message: r.message})
			continue
		}

		fileContent := r.content
		currentFileHash := hasher.ContentHash(fileContent)
		if currentFileHash == b.FileHash {
			result.FreshBlocks = append(result.FreshBlocks, *b)
			result.Fresh++
			continue
		}

		lines := strings.Split(fileContent, "\n")
		lineCount := b.SourceEnd - b.SourceStart + 1

		if b.SourceEnd <= len(lines) {
			selected := lines[b.SourceStart-1 : b.SourceEnd]
			if hasher.SnippetHash(selected) == b.SnippetHash {
				b.FileHash = currentFileHash
				result.FreshBlocks = append(result.FreshBlocks, *b)
				result.Fresh++
				needsRender = true
				continue
			}
		}

		start, end, found := scanner.FindSnippet(fileContent, b.SnippetHash, lineCount)
		if found {
			b.SourceStart = start
			b.SourceEnd = end
			b.FileHash = currentFileHash
			result.Healed++
			needsRender = true
			continue
		}

		result.Stale = append(result.Stale, StaleBlock{
			Block:   *b,
			Message: fmt.Sprintf("%s lines %d-%d: content changed", b.From, b.SourceStart, b.SourceEnd),
		})
	}

	if needsRender {
		result.Output = parser.Render(content, blocks)
	}

	return result, nil
}
