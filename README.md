# readme-merge

Keep README code examples in sync with real source code.

readme-merge embeds live code snippets into your README using HTML comment
placeholders. When your source changes, it detects staleness and updates the
examples — or blocks your commit until you do.

## Quick Start

Add a placeholder to your README that references a source file and line range:

```markdown
<!-- code from=src/client.go lines=10-15 -->
<!-- /code -->
```

Run `update` to populate it:

```
$ readme-merge update
updated 1 placeholder(s)
```

Your README now contains the live code, wrapped in a fenced block:

```markdown
<!-- code from=src/client.go lines=10-15 filehash=a1b2c3d4... snippethash=e5f6a7b8... -->
```go
client := NewClient(os.Getenv("API_KEY"))
resp, err := client.Send("hello")
if err != nil {
    log.Fatal(err)
}
```
<!-- /code -->
```

GitHub renders HTML comments as invisible — readers see only the code fence.

## Installation

### Linux / macOS

```
curl -fsSL https://raw.githubusercontent.com/phasecurve/readme-merge/main/install.sh | sh
```

Installs to `~/.local/bin` on Linux, `/usr/local/bin` on macOS. Override with `INSTALL_DIR`:

```
curl -fsSL https://raw.githubusercontent.com/phasecurve/readme-merge/main/install.sh | INSTALL_DIR=./bin sh
```

For private repo access, set `GITHUB_TOKEN`:

```
curl -fsSL -H "Authorization: token $GITHUB_TOKEN" \
  https://raw.githubusercontent.com/phasecurve/readme-merge/main/install.sh | sh
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/phasecurve/readme-merge/main/install.ps1 | iex
```

Installs to `%LOCALAPPDATA%\readme-merge` and offers to add it to your user PATH.
Override with `$env:INSTALL_DIR`. For private repo access, set `$env:GITHUB_TOKEN`.

### From source

```
go install github.com/phasecurve/readme-merge/cmd/readme-merge@latest
```

### Build locally

```
git clone https://github.com/phasecurve/readme-merge.git
cd readme-merge
go build -o readme-merge ./cmd/readme-merge
```

## Placeholder Syntax

Each placeholder is a pair of HTML comments:

```markdown
<!-- code from=<path> lines=<start>-<end> -->
<!-- /code -->
```

| Attribute | Required | Description |
|-----------|----------|-------------|
| `from`    | yes      | Relative path to the source file from the project root |
| `lines`   | yes      | Line range to extract, inclusive (e.g. `10-25`) |

After running `update`, two hash attributes are added automatically:

| Attribute      | Description |
|----------------|-------------|
| `filehash`     | SHA-256 hash of the entire source file (truncated to 16 hex chars) |
| `snippethash`  | SHA-256 hash of just the extracted lines |

These hashes are how readme-merge detects staleness. You never need to edit them.

You can place multiple placeholders in a single README, referencing different
files or different line ranges from the same file.

## Commands

### update

Populate or refresh all placeholders in the README:

```
readme-merge update [--source=<ref>] [--file=<path>]
```

Reads each referenced source file, extracts the specified lines, wraps them in a
fenced code block (with language detection from the file extension), and writes
the result back into the README. Hashes are computed and stored on each
placeholder.

### check

Verify that all placeholders are fresh:

```
readme-merge check [--source=<ref>] [--file=<path>] [--heal]
```

Exits 0 if everything is up to date. Exits 1 if any placeholder is stale or has
never been populated. By default, `check` is read-only and does not modify the
README. Pass `--heal` to write updated line references when snippets have shifted.

Output examples:

```
$ readme-merge check
2 placeholder(s) fresh

$ readme-merge check
1 stale placeholder(s):
  src/client.go lines 10-15: content changed
1 placeholder(s) fresh

$ readme-merge check
1 placeholder(s) have shifted lines (run with --heal to update)
1 placeholder(s) fresh

$ readme-merge check --heal
self-healed 1 placeholder(s) (lines shifted)
1 placeholder(s) fresh
```

### hook

Install or remove a git pre-commit hook:

```
readme-merge hook install
readme-merge hook uninstall
```

The hook runs `readme-merge check --source=staged --heal` before each commit.
Self-healing is enabled so that shifted line references are fixed automatically
as part of the commit. If any snippet is genuinely stale or unhashed, the commit
is blocked with output showing which placeholders need attention.

The hook is added as a clearly marked section in `.git/hooks/pre-commit`. If a
pre-commit hook already exists, readme-merge appends to it rather than
overwriting. Uninstall removes only the readme-merge section.

## Flags

### --source

Controls where source file content is read from:

| Value       | Reads from                          | Use case          |
|-------------|-------------------------------------|--------------------|
| *(default)* | Working tree files on disk          | Dev workflow       |
| `staged`    | Git index (`git show :path`)        | Pre-commit hook    |
| `HEAD`      | Last commit (`git show HEAD:path`)  | CI verification    |
| `<ref>`     | Any git ref (tag, branch, SHA)      | Pinned versions    |

The default reads files directly from disk. All other values require a git
repository.

### --file

Path to the README file. If omitted, readme-merge auto-detects by looking for
`README.md`, `readme.md`, or `Readme.md` in the current directory.

## How Staleness Detection Works

Each placeholder stores two hashes: one for the entire source file, one for just
the extracted snippet.

When `check` runs, it compares these hashes against the current source:

| File hash | Snippet at original lines | Scan finds snippet | Result |
|-----------|---------------------------|--------------------|--------|
| unchanged | -                         | -                  | Fresh (fast path) |
| changed   | matches                   | -                  | Fresh, file hash updated |
| changed   | no match                  | found elsewhere    | Self-healed: line numbers updated |
| changed   | no match                  | not found          | Stale: content genuinely changed |

The file hash is a quick first check that avoids unnecessary work when nothing
has changed.

### Self-Healing

When code moves (lines inserted or deleted above the referenced block) the line
numbers in the placeholder become wrong. Rather than immediately reporting
staleness, readme-merge scans the entire source file for a contiguous block whose
hash matches the stored snippet hash.

By default, `check` reports shifted placeholders without modifying the README:

```
$ readme-merge check
1 placeholder(s) have shifted lines (run with --heal to update)
2 placeholder(s) fresh
```

With `--heal`, the line references are updated in place:

```
$ readme-merge check --heal
self-healed 1 placeholder(s) (lines shifted)
2 placeholder(s) fresh
```

The pre-commit hook passes `--heal` automatically, so shifted references are
fixed as part of the commit with no developer intervention.

Self-healing only fires when the snippet content is identical but has moved. If
the content itself changed, the placeholder is reported as stale and the
developer must run `update` to review and accept the new code.

## Typical Workflow

**Initial setup:**

```
readme-merge hook install
```

**Writing documentation:**

1. Write your README with placeholders referencing real source files
2. Run `readme-merge update` to populate them
3. Commit both the README and source files

**Ongoing maintenance:**

When you edit source code that is referenced by the README, the pre-commit hook
catches it:

- If code moved but content is the same: the hook self-heals the line
  references automatically, commit proceeds.
- If code content changed: commit is blocked. Run `readme-merge update`, review
  the diff, then commit again.

**CI integration:**

Add `readme-merge check` to your CI pipeline to catch stale examples that slip
past the pre-commit hook:

```yaml
- name: Check README examples
  run: readme-merge check
```

## Security

Paths in `from` attributes are validated to prevent directory traversal. Absolute
paths and paths that escape the project directory (e.g. `../secret.txt`) are
rejected. Only relative paths within the project root are allowed.

## Supported Languages

Code fences are automatically annotated with the language based on the source
file extension:

| Extension | Language | Extension | Language |
|-----------|----------|-----------|----------|
| `.go`     | go       | `.py`     | python   |
| `.js`     | javascript | `.ts`   | typescript |
| `.rs`     | rust     | `.rb`     | ruby     |
| `.java`   | java     | `.sh`     | bash     |
| `.c`      | c        | `.cpp`    | cpp      |
| `.yaml`   | yaml     | `.json`   | json     |
| `.toml`   | toml     | `.sql`    | sql      |
| `.html`   | html     | `.css`    | css      |
| `.md`     | markdown |           |          |

Unrecognised extensions are used as-is.
