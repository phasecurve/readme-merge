package engine

import (
	"fmt"
	"os"
	"strings"

	"github.com/phasecurve/readme-merge/internal/hasher"
	"github.com/phasecurve/readme-merge/internal/parser"
	"github.com/phasecurve/readme-merge/internal/scanner"
	"github.com/phasecurve/readme-merge/internal/source"
)

type UpdateResult struct {
	Output  string
	Updated int
	Healed  int
}

type CheckResult struct {
	Stale    []StaleBlock
	Unhashed []parser.Block
	Healed   int
	Fresh    int
}

type StaleBlock struct {
	Block   parser.Block
	Message string
}

func Update(readmePath string, resolver *source.Resolver) (*UpdateResult, error) {
	data, err := os.ReadFile(readmePath)
	if err != nil {
		return nil, fmt.Errorf("reading README: %w", err)
	}

	content := string(data)
	blocks, err := parser.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("parsing README: %w", err)
	}

	result := &UpdateResult{}

	for i := range blocks {
		b := &blocks[i]

		fileContent, err := resolver.ReadFile(b.From)
		if err != nil {
			return nil, fmt.Errorf("block %s: %w", b.From, err)
		}

		lines := strings.Split(fileContent, "\n")
		if b.LineEnd > len(lines) || b.LineStart < 1 {
			return nil, fmt.Errorf("block %s: line range %d-%d out of bounds (%d lines)",
				b.From, b.LineStart, b.LineEnd, len(lines))
		}

		selected := lines[b.LineStart-1 : b.LineEnd]
		snippet := strings.Join(selected, "\n") + "\n"

		b.Content = snippet
		b.FileHash = hasher.ContentHash(fileContent)
		b.SnippetHash = hasher.ContentHash(snippet)
		result.Updated++
	}

	result.Output = parser.Render(content, blocks)
	if err := os.WriteFile(readmePath, []byte(result.Output), 0644); err != nil {
		return nil, fmt.Errorf("writing README: %w", err)
	}
	return result, nil
}

func Check(readmePath string, resolver *source.Resolver) (*CheckResult, error) {
	data, err := os.ReadFile(readmePath)
	if err != nil {
		return nil, fmt.Errorf("reading README: %w", err)
	}

	content := string(data)
	blocks, err := parser.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("parsing README: %w", err)
	}

	result := &CheckResult{}
	needsWrite := false

	for i := range blocks {
		b := &blocks[i]

		if b.FileHash == "" || b.SnippetHash == "" {
			result.Unhashed = append(result.Unhashed, *b)
			continue
		}

		fileContent, err := resolver.ReadFile(b.From)
		if err != nil {
			return nil, fmt.Errorf("block %s: %w", b.From, err)
		}

		currentFileHash := hasher.ContentHash(fileContent)
		if currentFileHash == b.FileHash {
			result.Fresh++
			continue
		}

		lines := strings.Split(fileContent, "\n")
		lineCount := b.LineEnd - b.LineStart + 1

		if b.LineEnd <= len(lines) {
			selected := lines[b.LineStart-1 : b.LineEnd]
			candidate := strings.Join(selected, "\n") + "\n"
			if hasher.ContentHash(candidate) == b.SnippetHash {
				b.FileHash = currentFileHash
				result.Fresh++
				needsWrite = true
				continue
			}
		}

		start, end, found := scanner.FindSnippet(fileContent, b.SnippetHash, lineCount)
		if found {
			b.LineStart = start
			b.LineEnd = end
			b.FileHash = currentFileHash
			result.Healed++
			needsWrite = true
			continue
		}

		result.Stale = append(result.Stale, StaleBlock{
			Block:   *b,
			Message: fmt.Sprintf("%s lines %d-%d: content changed", b.From, b.LineStart, b.LineEnd),
		})
	}

	if needsWrite {
		output := parser.Render(content, blocks)
		if err := os.WriteFile(readmePath, []byte(output), 0644); err != nil {
			return nil, fmt.Errorf("writing README: %w", err)
		}
	}

	return result, nil
}
