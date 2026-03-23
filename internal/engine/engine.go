package engine

import (
	"errors"
	"fmt"
	"strings"

	"github.com/phasecurve/readme-merge/internal/hasher"
	"github.com/phasecurve/readme-merge/internal/parser"
	"github.com/phasecurve/readme-merge/internal/remote"
	"github.com/phasecurve/readme-merge/internal/scanner"
)

type FileReader interface {
	ReadFile(path string, ref string) (string, error)
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

func Update(content string, reader FileReader) (*UpdateResult, error) {
	blocks, err := parser.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("parsing README: %w", err)
	}

	result := &UpdateResult{}
	fileCache := map[string]string{}

	for i := range blocks {
		b := &blocks[i]

		cacheKey := b.From + "\x00" + b.Ref
		fileContent, ok := fileCache[cacheKey]
		if !ok {
			var err error
			fileContent, err = reader.ReadFile(b.From, b.Ref)
			if err != nil {
				var refErr *remote.RefNotFoundError
				if errors.As(err, &refErr) {
					result.Unreachable = append(result.Unreachable, UnreachableBlock{
						Block:   *b,
						Message: refErr.Error(),
					})
					continue
				}
				return nil, fmt.Errorf("block %s: %w", b.From, err)
			}
			fileCache[cacheKey] = fileContent
		}

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
	fileCache := map[string]string{}

	for i := range blocks {
		b := &blocks[i]

		if b.FileHash == "" || b.SnippetHash == "" {
			result.Unhashed = append(result.Unhashed, *b)
			continue
		}

		cacheKey := b.From + "\x00" + b.Ref
		fileContent, ok := fileCache[cacheKey]
		if !ok {
			var err error
			fileContent, err = reader.ReadFile(b.From, b.Ref)
			if err != nil {
				var refErr *remote.RefNotFoundError
				if errors.As(err, &refErr) {
					result.Unreachable = append(result.Unreachable, UnreachableBlock{
						Block:   *b,
						Message: refErr.Error(),
					})
					continue
				}
				return nil, fmt.Errorf("block %s: %w", b.From, err)
			}
			fileCache[cacheKey] = fileContent
		}

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
