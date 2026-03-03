# readme-merge Design

A Go CLI tool that keeps README code examples in sync with real source code.

## Problem

Code examples in READMEs rot. They drift from the real implementation, mislead
users, and nobody notices until someone tries to use them. Manual copy-paste
doesn't scale and is forgotten.

## Solution

Placeholders in the README reference real source files by path and line range.
The tool extracts the code, injects it into the README, and tracks changes with
a two-hash system that self-heals when code moves and warns when code changes.

## Placeholder Format

Dev writes:

```markdown
<!-- code from=examples/client.go lines=10-25 -->
<!-- /code -->
```

After `readme-merge update`:

```markdown
<!-- code from=examples/client.go lines=10-25 filehash=abc123 snippethash=def456 -->
```go
client := NewClient(os.Getenv("API_KEY"))
resp, err := client.Send("hello")
if err != nil {
    log.Fatal(err)
}
```
<!-- /code -->
```

The opening comment stores metadata. The closing `<!-- /code -->` marks the end
of the injected block. GitHub renders HTML comments as invisible, so readers see
only the code fence.

## Two-Hash Staleness Detection

Each placeholder tracks two hashes (SHA-256, truncated to 16 hex chars):

- **File hash**: hash of the entire source file content. Quick first check.
- **Snippet hash**: hash of just the extracted lines. Does the real work.

Detection logic:

| File hash   | Snippet at original lines | Scan finds snippet | Result                                       |
|-------------|---------------------------|--------------------|----------------------------------------------|
| unchanged   | -                         | -                  | Skip (fast path)                             |
| changed     | matches                   | -                  | Update file hash only                        |
| changed     | no match                  | found elsewhere    | Self-heal: update lines + file hash silently |
| changed     | no match                  | not found          | WARN: content changed, dev must review       |

Self-healing: when code shifts position (lines inserted/deleted above), the tool
scans the file for a contiguous block whose hash matches the stored snippet hash.
If found, it silently updates the line reference. No dev intervention needed for
the common case of code moving around.

## CLI Commands

```
readme-merge update                     # populate/refresh all placeholders (worktree)
readme-merge update --source=staged     # from git staged files
readme-merge update --source=HEAD       # from last commit
readme-merge update --source=v2.1.0     # from specific git ref

readme-merge check                      # exit 0 if fresh, exit 1 if stale/unhashed
readme-merge check --source=staged      # for pre-commit hook use

readme-merge hook install               # install git pre-commit hook
readme-merge hook uninstall             # remove hook
```

### check mode behaviour

- Stale snippet (content changed): exit 1, prints which placeholders are stale
  with file path and line range.
- Unhashed placeholder (first run, never populated): exit 1, tells dev to run
  `update`.
- Self-healed (lines shifted, content identical): exit 0, silently updates line
  numbers in README.
- All fresh: exit 0.

## --source Flag

Controls where source file content is read from.

| Value       | Reads from                          | Use case          |
|-------------|-------------------------------------|--------------------|
| *(default)* | Working tree files on disk          | Dev workflow       |
| `staged`    | Git index (`git show :path`)        | Pre-commit hook    |
| `HEAD`      | Last commit (`git show HEAD:path`)  | CI verification    |
| `<ref>`     | Any git ref (tag, branch, SHA)      | Pinned versions    |

When not in a git repo, only the default (worktree) works. Other values produce
a clear error: "--source=staged requires a git repository".

## Pre-commit Hook

`readme-merge hook install` writes `.git/hooks/pre-commit` (or appends to an
existing one) that runs:

```bash
readme-merge check --source=staged
```

If any snippet is stale or unhashed, the commit is blocked with output showing
which placeholders need attention.

`readme-merge hook uninstall` removes the readme-merge section from the hook.

## Project Structure

```
~/dev/readme-merge/
├── cmd/
│   └── readme-merge/
│       └── main.go          # CLI entry point, cobra commands
├── internal/
│   ├── parser/              # parse README, find/update <!-- code --> blocks
│   ├── extractor/           # read source files, extract line ranges
│   ├── hasher/              # SHA-256 truncated, file + snippet hashing
│   ├── scanner/             # self-healing: scan file for relocated snippet
│   ├── source/              # --source resolution: worktree vs git ref
│   └── hook/                # install/uninstall pre-commit hook
├── go.mod
└── go.sum
```

## Decisions

- **Line ranges, not source code markers**: placeholders reference file:lines
  only. No comments injected into source code. The self-healing hash scan
  handles the fragility of line numbers.
- **Content hash, not git object hash**: hash the extracted text, not the git
  blob. Works without git, detects only actual content changes (not unrelated
  file edits).
- **Inline HTML comments, not template files**: README.md is both template and
  output. No `.tmpl` file to forget about. HTML comments are invisible on GitHub.
- **Strict check mode**: unhashed placeholders are errors, not auto-populated.
  Dev must consciously run `update` first.
