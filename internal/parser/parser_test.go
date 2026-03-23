package parser_test

import (
	"strings"
	"testing"

	"github.com/phasecurve/readme-merge/internal/parser"
)

func TestParseNewPlaceholder(t *testing.T) {
	input := "# Title\n\n<!-- code from=examples/client.go lines=10-25 -->\n<!-- /code -->\n\nMore text.\n"

	blocks, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	b := blocks[0]
	if b.From != "examples/client.go" {
		t.Errorf("From = %q, want %q", b.From, "examples/client.go")
	}
	if b.SourceStart != 10 || b.SourceEnd != 25 {
		t.Errorf("Lines = %d-%d, want 10-25", b.SourceStart, b.SourceEnd)
	}
	if b.FileHash != "" || b.SnippetHash != "" {
		t.Errorf("new placeholder should have empty hashes")
	}
	if b.Content != "" {
		t.Errorf("new placeholder should have empty content")
	}
}

func TestParsePopulatedPlaceholder(t *testing.T) {
	input := "<!-- code from=src/main.go lines=1-3 filehash=aaaa5678 snippethash=bbbb5678 -->\n```go\npackage main\n```\n<!-- /code -->\n"

	blocks, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	b := blocks[0]
	if b.FileHash != "aaaa5678" {
		t.Errorf("FileHash = %q, want %q", b.FileHash, "aaaa5678")
	}
	if b.SnippetHash != "bbbb5678" {
		t.Errorf("SnippetHash = %q, want %q", b.SnippetHash, "bbbb5678")
	}
	if b.Content != "package main\n" {
		t.Errorf("Content = %q, want %q", b.Content, "package main\n")
	}
}

func TestParseMultiplePlaceholders(t *testing.T) {
	input := "<!-- code from=a.go lines=1-2 -->\n<!-- /code -->\n\nText\n\n<!-- code from=b.go lines=5-10 -->\n<!-- /code -->\n"

	blocks, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
	if blocks[0].From != "a.go" || blocks[1].From != "b.go" {
		t.Errorf("wrong file refs: %q, %q", blocks[0].From, blocks[1].From)
	}
}

func TestParseCrossRepoWithRef(t *testing.T) {
	input := "<!-- code from=git@github.com:org/repo.git//README.md ref=main lines=30-45 -->\n<!-- /code -->\n"

	blocks, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	b := blocks[0]
	if b.From != "git@github.com:org/repo.git//README.md" {
		t.Errorf("From = %q, want %q", b.From, "git@github.com:org/repo.git//README.md")
	}
	if b.Ref != "main" {
		t.Errorf("Ref = %q, want %q", b.Ref, "main")
	}
	if b.SourceStart != 30 || b.SourceEnd != 45 {
		t.Errorf("Lines = %d-%d, want 30-45", b.SourceStart, b.SourceEnd)
	}
}

func TestParseCrossRepoWithoutRef(t *testing.T) {
	input := "<!-- code from=git@github.com:org/repo.git//README.md lines=1-10 -->\n<!-- /code -->\n"

	blocks, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	b := blocks[0]
	if b.From != "git@github.com:org/repo.git//README.md" {
		t.Errorf("From = %q, want %q", b.From, "git@github.com:org/repo.git//README.md")
	}
	if b.Ref != "" {
		t.Errorf("Ref = %q, want empty", b.Ref)
	}
	if b.SourceStart != 1 || b.SourceEnd != 10 {
		t.Errorf("Lines = %d-%d, want 1-10", b.SourceStart, b.SourceEnd)
	}
}

func TestParseCrossRepoWithRefContainingSlash(t *testing.T) {
	input := "<!-- code from=git@github.com:org/repo.git//docs/guide.md ref=chore/clean-docs lines=5-20 -->\n<!-- /code -->\n"

	blocks, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	b := blocks[0]
	if b.From != "git@github.com:org/repo.git//docs/guide.md" {
		t.Errorf("From = %q, want %q", b.From, "git@github.com:org/repo.git//docs/guide.md")
	}
	if b.Ref != "chore/clean-docs" {
		t.Errorf("Ref = %q, want %q", b.Ref, "chore/clean-docs")
	}
}

func TestRenderNewBlock(t *testing.T) {
	original := "# Title\n\n<!-- code from=examples/client.go lines=10-12 -->\n<!-- /code -->\n\nMore text.\n"

	blocks, _ := parser.Parse(original)
	blocks[0].Content = "fmt.Println(\"hello\")\n"
	blocks[0].FileHash = "aaaa1234aaaa1234"
	blocks[0].SnippetHash = "bbbb1234bbbb1234"
	blocks[0].SourceStart = 10
	blocks[0].SourceEnd = 12

	got := parser.Render(original, blocks)

	want := "# Title\n\n<!-- code from=examples/client.go lines=10-12 filehash=aaaa1234aaaa1234 snippethash=bbbb1234bbbb1234 -->\n```go\nfmt.Println(\"hello\")\n```\n<!-- /code -->\n\nMore text.\n"
	if got != want {
		t.Errorf("Render mismatch.\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderCrossRepoBlockIncludesRef(t *testing.T) {
	original := "<!-- code from=git@github.com:org/repo.git//README.md ref=main lines=1-3 -->\n<!-- /code -->\n"

	blocks, _ := parser.Parse(original)
	blocks[0].Content = "some content\n"
	blocks[0].FileHash = "aaaa1234aaaa1234"
	blocks[0].SnippetHash = "bbbb1234bbbb1234"

	got := parser.Render(original, blocks)

	want := "<!-- code from=git@github.com:org/repo.git//README.md ref=main lines=1-3 filehash=aaaa1234aaaa1234 snippethash=bbbb1234bbbb1234 -->\n```markdown\nsome content\n```\n<!-- /code -->\n"
	if got != want {
		t.Errorf("Render mismatch.\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderLocalBlockOmitsRef(t *testing.T) {
	original := "<!-- code from=src/main.go lines=1-3 -->\n<!-- /code -->\n"

	blocks, _ := parser.Parse(original)
	blocks[0].Content = "package main\n"
	blocks[0].FileHash = "aaaa1234aaaa1234"
	blocks[0].SnippetHash = "bbbb1234bbbb1234"

	got := parser.Render(original, blocks)

	if strings.Contains(got, "ref=") {
		t.Errorf("local block should not contain ref= in output, got:\n%s", got)
	}
}

func TestRenderEscapesNestedCodeFences(t *testing.T) {
	original := "<!-- code from=docs/example.md lines=1-3 -->\n<!-- /code -->\n"

	blocks, _ := parser.Parse(original)
	blocks[0].Content = "Some text\n```yaml\nkey: value\n```\nMore text\n"
	blocks[0].FileHash = "aaaa1234aaaa1234"
	blocks[0].SnippetHash = "bbbb1234bbbb1234"

	got := parser.Render(original, blocks)

	if !strings.Contains(got, "````markdown") {
		t.Errorf("expected 4-backtick fence to escape inner 3-backtick fence, got:\n%s", got)
	}
	if !strings.Contains(got, "````\n<!-- /code -->") {
		t.Errorf("expected closing 4-backtick fence, got:\n%s", got)
	}
}

func TestRenderEscapesDeeplyNestedFences(t *testing.T) {
	original := "<!-- code from=docs/example.md lines=1-3 -->\n<!-- /code -->\n"

	blocks, _ := parser.Parse(original)
	blocks[0].Content = "````python\ncode\n````\n"
	blocks[0].FileHash = "aaaa1234aaaa1234"
	blocks[0].SnippetHash = "bbbb1234bbbb1234"

	got := parser.Render(original, blocks)

	if !strings.Contains(got, "`````markdown") {
		t.Errorf("expected 5-backtick fence to escape inner 4-backtick fence, got:\n%s", got)
	}
}

func TestParseIslandSingleRange(t *testing.T) {
	input := "# Title\n\n<!-- island file=\"docs/guide.md\" -->\n<!-- lines from=\"10\" to=\"14\" -->\n<!-- end island -->\n\nMore text.\n"

	blocks, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	b := blocks[0]
	if b.From != "docs/guide.md" {
		t.Errorf("From = %q, want %q", b.From, "docs/guide.md")
	}
	if b.SourceStart != 10 || b.SourceEnd != 14 {
		t.Errorf("Lines = %d-%d, want 10-14", b.SourceStart, b.SourceEnd)
	}
	if b.IslandID == "" {
		t.Error("expected non-empty IslandID")
	}
	if b.IslandIndex != 0 {
		t.Errorf("IslandIndex = %d, want 0", b.IslandIndex)
	}
	if b.IslandTotal != 1 {
		t.Errorf("IslandTotal = %d, want 1", b.IslandTotal)
	}
	if b.Render != parser.RenderRaw {
		t.Errorf("Render = %q, want %q", b.Render, parser.RenderRaw)
	}
}

func TestParseIslandMultipleRanges(t *testing.T) {
	input := "<!-- island file=\"docs/guide.md\" -->\n<!-- lines from=\"10\" to=\"14\" -->\n<!-- lines from=\"54\" to=\"62\" -->\n<!-- end island -->\n"

	blocks, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}

	if blocks[0].IslandID != blocks[1].IslandID {
		t.Error("sub-blocks should share IslandID")
	}
	if blocks[0].IslandIndex != 0 || blocks[1].IslandIndex != 1 {
		t.Errorf("IslandIndex = %d, %d; want 0, 1", blocks[0].IslandIndex, blocks[1].IslandIndex)
	}
	if blocks[0].IslandTotal != 2 || blocks[1].IslandTotal != 2 {
		t.Errorf("IslandTotal = %d, %d; want 2, 2", blocks[0].IslandTotal, blocks[1].IslandTotal)
	}
	if blocks[0].SourceStart != 10 || blocks[0].SourceEnd != 14 {
		t.Errorf("block[0] Lines = %d-%d, want 10-14", blocks[0].SourceStart, blocks[0].SourceEnd)
	}
	if blocks[1].SourceStart != 54 || blocks[1].SourceEnd != 62 {
		t.Errorf("block[1] Lines = %d-%d, want 54-62", blocks[1].SourceStart, blocks[1].SourceEnd)
	}
}

func TestParseIslandWithRepoAndRef(t *testing.T) {
	input := "<!-- island file=\"README.md\" repo=\"git@github.com:org/repo.git\" ref=\"main\" -->\n<!-- lines from=\"1\" to=\"5\" -->\n<!-- end island -->\n"

	blocks, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	b := blocks[0]
	if b.From != "git@github.com:org/repo.git//README.md" {
		t.Errorf("From = %q, want %q", b.From, "git@github.com:org/repo.git//README.md")
	}
	if b.Ref != "main" {
		t.Errorf("Ref = %q, want %q", b.Ref, "main")
	}
}

func TestParseIslandWithSnippetHashes(t *testing.T) {
	input := "<!-- island file=\"docs/guide.md\" filehash=aaa111 -->\n<!-- lines from=\"10\" to=\"14\" snippethash=bbb222 -->\n<!-- lines from=\"54\" to=\"62\" snippethash=ccc333 -->\n<!-- end island -->\n"

	blocks, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}

	if blocks[0].FileHash != "aaa111" {
		t.Errorf("FileHash = %q, want %q", blocks[0].FileHash, "aaa111")
	}
	if blocks[0].SnippetHash != "bbb222" {
		t.Errorf("block[0] SnippetHash = %q, want %q", blocks[0].SnippetHash, "bbb222")
	}
	if blocks[1].SnippetHash != "ccc333" {
		t.Errorf("block[1] SnippetHash = %q, want %q", blocks[1].SnippetHash, "ccc333")
	}
}

func TestParseIslandUnclosedError(t *testing.T) {
	input := "<!-- island file=\"docs/guide.md\" -->\n<!-- lines from=\"10\" to=\"14\" -->\n"

	_, err := parser.Parse(input)
	if err == nil {
		t.Fatal("expected error for unclosed island")
	}
	if !strings.Contains(err.Error(), "unclosed") {
		t.Errorf("expected 'unclosed' in error, got: %v", err)
	}
}

func TestParseIslandNoLinesError(t *testing.T) {
	input := "<!-- island file=\"docs/guide.md\" -->\n<!-- end island -->\n"

	_, err := parser.Parse(input)
	if err == nil {
		t.Fatal("expected error for island with no <lines> elements")
	}
}

func TestRenderIslandRaw(t *testing.T) {
	original := "# Title\n\n<!-- island file=\"docs/guide.md\" -->\n<!-- lines from=\"10\" to=\"14\" -->\n<!-- end island -->\n\nMore text.\n"

	blocks, _ := parser.Parse(original)
	blocks[0].Content = "This is raw content\nfrom the guide.\n"
	blocks[0].FileHash = "aaaa1234"
	blocks[0].SnippetHash = "bbbb1234"

	got := parser.Render(original, blocks)

	if strings.Contains(got, "```") {
		t.Errorf("island should render raw (no fences), got:\n%s", got)
	}
	if !strings.Contains(got, "This is raw content") {
		t.Errorf("expected raw content in output, got:\n%s", got)
	}
	if !strings.Contains(got, "<!-- island file=\"docs/guide.md\"") {
		t.Errorf("expected island opening comment, got:\n%s", got)
	}
	if !strings.Contains(got, "<!-- end island -->") {
		t.Errorf("expected island closing comment, got:\n%s", got)
	}
	if !strings.Contains(got, "filehash=aaaa1234") {
		t.Errorf("expected filehash in island header, got:\n%s", got)
	}
	if !strings.Contains(got, "snippethash=bbbb1234") {
		t.Errorf("expected snippethash on <lines> tag, got:\n%s", got)
	}
}

func TestRenderIslandMultipleRanges(t *testing.T) {
	original := "<!-- island file=\"docs/guide.md\" -->\n<!-- lines from=\"10\" to=\"12\" -->\n<!-- lines from=\"54\" to=\"56\" -->\n<!-- end island -->\n"

	blocks, _ := parser.Parse(original)
	blocks[0].Content = "line ten\nline eleven\nline twelve\n"
	blocks[0].FileHash = "aaaa1234"
	blocks[0].SnippetHash = "bbbb1234"
	blocks[1].Content = "line fiftyfour\nline fiftyfive\nline fiftysix\n"
	blocks[1].FileHash = "aaaa1234"
	blocks[1].SnippetHash = "cccc1234"

	got := parser.Render(original, blocks)

	if !strings.Contains(got, "line ten") {
		t.Errorf("expected first range content, got:\n%s", got)
	}
	if !strings.Contains(got, "line fiftyfour") {
		t.Errorf("expected second range content, got:\n%s", got)
	}
	if strings.Count(got, "<!-- island") != 1 {
		t.Errorf("expected exactly one island opening, got:\n%s", got)
	}
	if strings.Count(got, "<!-- end island -->") != 1 {
		t.Errorf("expected exactly one island closing, got:\n%s", got)
	}
}

func TestRenderMixedCodeAndIsland(t *testing.T) {
	original := "<!-- code from=src/main.go lines=1-1 -->\n<!-- /code -->\n\n<!-- island file=\"docs/guide.md\" -->\n<!-- lines from=\"1\" to=\"2\" -->\n<!-- end island -->\n"

	blocks, _ := parser.Parse(original)
	blocks[0].Content = "package main\n"
	blocks[0].FileHash = "aaaa1234"
	blocks[0].SnippetHash = "bbbb1234"
	blocks[1].Content = "guide content\n"
	blocks[1].FileHash = "cccc1234"
	blocks[1].SnippetHash = "dddd1234"

	got := parser.Render(original, blocks)

	if !strings.Contains(got, "```go") {
		t.Errorf("code block should have fences, got:\n%s", got)
	}
	if !strings.Contains(got, "guide content") {
		t.Errorf("island should have raw content, got:\n%s", got)
	}
}

func TestParseIslandRoundTrip(t *testing.T) {
	input := "<!-- island file=\"docs/guide.md\" filehash=aaa111 -->\n" +
		"<!-- lines from=\"10\" to=\"12\" snippethash=bbb222 -->\n" +
		"line ten\nline eleven\nline twelve\n" +
		"<!-- lines from=\"54\" to=\"56\" snippethash=ccc333 -->\n" +
		"line fiftyfour\nline fiftyfive\nline fiftysix\n" +
		"<!-- end island -->\n"

	blocks, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}

	if blocks[0].Content != "line ten\nline eleven\nline twelve\n" {
		t.Errorf("block[0] Content = %q, want %q", blocks[0].Content, "line ten\nline eleven\nline twelve\n")
	}
	if blocks[1].Content != "line fiftyfour\nline fiftyfive\nline fiftysix\n" {
		t.Errorf("block[1] Content = %q, want %q", blocks[1].Content, "line fiftyfour\nline fiftyfive\nline fiftysix\n")
	}
	if blocks[0].SnippetHash != "bbb222" {
		t.Errorf("block[0] SnippetHash = %q, want %q", blocks[0].SnippetHash, "bbb222")
	}
	if blocks[1].SnippetHash != "ccc333" {
		t.Errorf("block[1] SnippetHash = %q, want %q", blocks[1].SnippetHash, "ccc333")
	}
}

func TestRewriteAnchorLinksCrossRepoIsland(t *testing.T) {
	original := "<!-- island file=\"docs/guide.md\" repo=\"git@github.com:org/repo.git\" ref=\"main\" -->\n<!-- lines from=\"12\" to=\"14\" -->\n<!-- end island -->\n"

	blocks, _ := parser.Parse(original)
	blocks[0].Content = "- [Why uv](#11-why-uv)\n- [Installation](#12-installation)\n"
	blocks[0].FileHash = "aaaa1234"
	blocks[0].SnippetHash = "bbbb1234"

	got := parser.Render(original, blocks)

	if !strings.Contains(got, "https://github.com/org/repo/blob/main/docs/guide.md#11-why-uv") {
		t.Errorf("expected rewritten anchor link, got:\n%s", got)
	}
	if !strings.Contains(got, "https://github.com/org/repo/blob/main/docs/guide.md#12-installation") {
		t.Errorf("expected rewritten anchor link for second item, got:\n%s", got)
	}
	if strings.Contains(got, "](#") {
		t.Errorf("should not have bare anchor links remaining, got:\n%s", got)
	}
}

func TestRewriteAnchorLinksCrossRepoCode(t *testing.T) {
	original := "<!-- code from=git@github.com:org/repo.git//docs/guide.md ref=main lines=12-14 -->\n<!-- /code -->\n"

	blocks, _ := parser.Parse(original)
	blocks[0].Content = "- [Why uv](#11-why-uv)\n"
	blocks[0].FileHash = "aaaa1234"
	blocks[0].SnippetHash = "bbbb1234"

	got := parser.Render(original, blocks)

	if !strings.Contains(got, "https://github.com/org/repo/blob/main/docs/guide.md#11-why-uv") {
		t.Errorf("expected rewritten anchor link in code block, got:\n%s", got)
	}
}

func TestNoRewriteLocalIsland(t *testing.T) {
	original := "<!-- island file=\"docs/guide.md\" -->\n<!-- lines from=\"1\" to=\"2\" -->\n<!-- end island -->\n"

	blocks, _ := parser.Parse(original)
	blocks[0].Content = "- [Section](#section)\n"
	blocks[0].FileHash = "aaaa1234"
	blocks[0].SnippetHash = "bbbb1234"

	got := parser.Render(original, blocks)

	if !strings.Contains(got, "](#section)") {
		t.Errorf("local island should not rewrite anchor links, got:\n%s", got)
	}
}

func TestRewriteAnchorLinksSSHFormat(t *testing.T) {
	original := "<!-- island file=\"docs/guide.md\" repo=\"git@github.com:org/repo.git\" ref=\"v2.0\" -->\n<!-- lines from=\"1\" to=\"1\" -->\n<!-- end island -->\n"

	blocks, _ := parser.Parse(original)
	blocks[0].Content = "- [Intro](#intro)\n"
	blocks[0].FileHash = "aaaa1234"
	blocks[0].SnippetHash = "bbbb1234"

	got := parser.Render(original, blocks)

	if !strings.Contains(got, "https://github.com/org/repo/blob/v2.0/docs/guide.md#intro") {
		t.Errorf("expected correct blob URL with ref, got:\n%s", got)
	}
}

func TestRewriteAnchorLinksHTTPSFormat(t *testing.T) {
	original := "<!-- island file=\"docs/guide.md\" repo=\"https://github.com/org/repo.git\" ref=\"main\" -->\n<!-- lines from=\"1\" to=\"1\" -->\n<!-- end island -->\n"

	blocks, _ := parser.Parse(original)
	blocks[0].Content = "- [Intro](#intro)\n"
	blocks[0].FileHash = "aaaa1234"
	blocks[0].SnippetHash = "bbbb1234"

	got := parser.Render(original, blocks)

	if !strings.Contains(got, "https://github.com/org/repo/blob/main/docs/guide.md#intro") {
		t.Errorf("expected correct blob URL from HTTPS repo, got:\n%s", got)
	}
}

func TestRewritePreservesNonAnchorLinks(t *testing.T) {
	original := "<!-- island file=\"docs/guide.md\" repo=\"git@github.com:org/repo.git\" ref=\"main\" -->\n<!-- lines from=\"1\" to=\"2\" -->\n<!-- end island -->\n"

	blocks, _ := parser.Parse(original)
	blocks[0].Content = "- [External](https://example.com)\n- [Anchor](#section)\n"
	blocks[0].FileHash = "aaaa1234"
	blocks[0].SnippetHash = "bbbb1234"

	got := parser.Render(original, blocks)

	if !strings.Contains(got, "(https://example.com)") {
		t.Errorf("should preserve external links, got:\n%s", got)
	}
	if strings.Contains(got, "](#section)") {
		t.Errorf("should rewrite anchor link, got:\n%s", got)
	}
}

func TestRenderIslandMarkupIsHidden(t *testing.T) {
	original := "<!-- island file=\"docs/guide.md\" -->\n<!-- lines from=\"10\" to=\"12\" -->\n<!-- lines from=\"54\" to=\"56\" -->\n<!-- end island -->\n"

	blocks, _ := parser.Parse(original)
	blocks[0].Content = "first range content\n"
	blocks[0].FileHash = "aaaa1234"
	blocks[0].SnippetHash = "bbbb1234"
	blocks[1].Content = "second range content\n"
	blocks[1].FileHash = "aaaa1234"
	blocks[1].SnippetHash = "cccc1234"

	got := parser.Render(original, blocks)

	for _, line := range strings.Split(got, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		isContent := !strings.HasPrefix(trimmed, "<!--")
		isComment := strings.HasPrefix(trimmed, "<!--") && strings.HasSuffix(trimmed, "-->")
		if !isContent && !isComment {
			t.Errorf("line is neither pure content nor a valid HTML comment (would be visible on GitHub): %q", line)
		}
		if strings.HasPrefix(trimmed, "<") && !strings.HasPrefix(trimmed, "<!--") {
			t.Errorf("line starts with HTML tag that GitHub may render or strip: %q", line)
		}
	}
}

func TestRenderIslandAllTagsAreComments(t *testing.T) {
	original := "<!-- island file=\"docs/guide.md\" repo=\"git@github.com:org/repo.git\" ref=\"main\" -->\n<!-- lines from=\"1\" to=\"2\" -->\n<!-- end island -->\n"

	blocks, _ := parser.Parse(original)
	blocks[0].Content = "some content\n"
	blocks[0].FileHash = "aaaa1234"
	blocks[0].SnippetHash = "bbbb1234"

	got := parser.Render(original, blocks)

	for _, line := range strings.Split(got, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || !strings.HasPrefix(trimmed, "<") {
			continue
		}
		if !strings.HasPrefix(trimmed, "<!--") || !strings.HasSuffix(trimmed, "-->") {
			t.Errorf("HTML-like line is not a comment (visible on GitHub): %q", line)
		}
	}
}

func TestRenderNoEscapeWhenNoNestedFences(t *testing.T) {
	original := "<!-- code from=src/main.go lines=1-1 -->\n<!-- /code -->\n"

	blocks, _ := parser.Parse(original)
	blocks[0].Content = "package main\n"
	blocks[0].FileHash = "aaaa1234aaaa1234"
	blocks[0].SnippetHash = "bbbb1234bbbb1234"

	got := parser.Render(original, blocks)

	if strings.Contains(got, "````") {
		t.Errorf("should use 3-backtick fence when no nesting needed, got:\n%s", got)
	}
	if !strings.Contains(got, "```go") {
		t.Errorf("expected normal 3-backtick fence, got:\n%s", got)
	}
}

func TestParseRoundTripsEscapedFences(t *testing.T) {
	input := "<!-- code from=docs/example.md lines=1-5 filehash=aaaa snippethash=bbbb -->\n````markdown\nSome text\n```yaml\nkey: value\n```\nMore text\n````\n<!-- /code -->\n"

	blocks, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	b := blocks[0]
	want := "Some text\n```yaml\nkey: value\n```\nMore text\n"
	if b.Content != want {
		t.Errorf("Content = %q, want %q", b.Content, want)
	}
}

func TestRenderPreservesUnchangedText(t *testing.T) {
	original := "Line 1\nLine 2\n<!-- code from=a.txt lines=1-1 -->\n<!-- /code -->\nLine 5\n"

	blocks, _ := parser.Parse(original)
	blocks[0].Content = "hello\n"
	blocks[0].FileHash = "aaaa1234aaaa1234"
	blocks[0].SnippetHash = "bbbb1234bbbb1234"

	got := parser.Render(original, blocks)

	if !strings.HasPrefix(got, "Line 1\nLine 2\n") {
		t.Errorf("text before block was modified")
	}
	if !strings.HasSuffix(got, "Line 5\n") {
		t.Errorf("text after block was modified")
	}
}
