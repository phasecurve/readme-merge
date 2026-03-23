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

GitHub renders HTML comments as invisible; readers see only the code fence.

You can also reference files in other git repositories:

```markdown
<!-- code from=git@github.com:org/shared-lib.git//README.md ref=main lines=10-25 -->
<!-- /code -->
```

See [Cross-Repository References](#cross-repository-references) for details.

For raw content (no code fences), use [Islands](#islands):

```markdown
<!-- island file="docs/guide.md" -->
<!-- lines from="10" to="14" -->
<!-- end island -->
```

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
| `from`    | yes      | Relative path to the source file from the project root, or a cross-repo reference (see below) |
| `ref`     | no       | Git ref (branch, tag, SHA) for cross-repo references. Defaults to `main`. Ignored for local files. |
| `lines`   | yes      | Line range to extract, inclusive (e.g. `10-25`) |

After running `update`, two hash attributes are added automatically:

| Attribute      | Description |
|----------------|-------------|
| `filehash`     | SHA-256 hash of the entire source file (truncated to 16 hex chars) |
| `snippethash`  | SHA-256 hash of just the extracted lines |

These hashes are how readme-merge detects staleness. You never need to edit them.

You can place multiple placeholders in a single README, referencing different
files or different line ranges from the same file.

To embed raw content without code fences (useful for markdown prose, tables of
contents, or documentation fragments), see [Islands](#islands).

## Cross-Repository References

You can embed code from other git repositories by using `//` to separate the
repo URL from the file path:

```markdown
<!-- code from=git@github.com:org/shared-lib.git//README.md ref=main lines=10-25 -->
<!-- /code -->
```

The `from` value has the format `<git-url>//<file-path>`. Everything before `//`
is the repository URL (SSH or HTTPS). Everything after is the path within that
repository. The `ref` attribute specifies which branch, tag, or commit to read
from. If omitted, it defaults to `main`.

Use a long-lived ref (e.g. `main`, a release tag like `v2.0`, or a SHA). If a
ref points to a feature branch that is later deleted, readme-merge will warn
that the ref is unreachable and skip the block, preserving whatever content is
already in the README. Update the `ref` attribute to a valid branch or tag to
resolve the warning.

Both SSH and HTTPS URLs are supported:

```markdown
<!-- code from=git@github.com:org/repo.git//src/config.py ref=v2.0 lines=1-10 -->
<!-- /code -->

<!-- code from=https://github.com/org/repo.git//docs/guide.md lines=5-20 -->
<!-- /code -->
```

### How it works

Remote repositories are cached as bare git repos in `.readme-merge/cache/`,
named after the repository:

```
.readme-merge/cache/
  shared-lib/       # bare git repo
  other-repo/       # bare git repo
```

On the first run, readme-merge initialises a bare clone and fetches the
requested ref. On subsequent runs, it fetches only the diff. Files are read
directly from the git object store via `git show`, so no working tree is
created.

Add `.readme-merge/` to your `.gitignore`:

```
# readme-merge cache
.readme-merge/
```

### Staleness and self-healing

Cross-repo blocks use the same two-hash staleness detection as local blocks.
When `check` or `update` runs, it fetches the latest state of the pinned ref
and compares hashes. Self-healing works the same way: if the referenced lines
shift in the remote file, readme-merge finds the snippet by hash and updates
the line numbers.

The `--source` flag (staged, HEAD, etc.) applies only to local blocks.
Cross-repo blocks always resolve against their own `ref`.

## Islands

Islands embed raw content from another file (typically markdown) without wrapping
it in a code fence. This is useful for pulling prose sections, tables of
contents, or documentation fragments from other files into your README.

### Syntax

```markdown
<!-- island file="docs/guide.md" -->
<!-- lines from="10" to="14" -->
<!-- lines from="54" to="62" -->
<!-- end island -->
```

Each `<!-- lines -->` element specifies a line range to extract. You can include as
many ranges as you need; they are extracted independently.

| Attribute | On | Required | Description |
|-----------|----|----------|-------------|
| `file`    | `island` | yes | Path to the source file |
| `repo`    | `island` | no  | Git repo URL for cross-repo references |
| `ref`     | `island` | no  | Git ref (branch, tag, SHA). Defaults to `main` for cross-repo. |
| `filehash`| `island` | no  | Added automatically by `update` |
| `from`    | `lines`  | yes | Start line (inclusive) |
| `to`      | `lines`  | yes | End line (inclusive) |
| `snippethash` | `lines` | no | Added automatically by `update`, per range |

After running `update`, hashes are added and raw content appears between each
`<!-- lines -->` tag:

```markdown
<!-- island file="docs/guide.md" filehash=a1b2c3d4... -->
<!-- lines from="10" to="14" snippethash=e5f6a7b8... -->
This content is spliced in raw,
not wrapped in a code fence.
<!-- lines from="54" to="62" snippethash=c9d0e1f2... -->
More raw content from a different
section of the same file.
<!-- end island -->
```

GitHub renders HTML comments as invisible. Readers see only the raw content.

### Cross-repo islands

Islands support the same cross-repo references as code blocks:

```markdown
<!-- island file="docs/python-style.md" repo="git@github.com:org/standards.git" ref="main" -->
<!-- lines from="12" to="30" -->
<!-- end island -->
```

The `repo` and `ref` attributes work identically to the `from` and `ref`
attributes on code blocks. The repo is cached as a bare clone in
`.readme-merge/cache/`.

### Anchor link rewriting

When pulling markdown content from a cross-repo source, any anchor links
(`](#section-name)`) are automatically rewritten to full GitHub URLs so they
point back to the correct location in the source file:

```
Before: [Installation](#12-installation)
After:  [Installation](https://github.com/org/standards/blob/main/docs/python-style.md#12-installation)
```

This applies to both islands and code blocks with cross-repo references.
Local references are left unchanged.

### Self-healing

Each `<!-- lines -->` range is tracked independently with its own snippet hash. If
lines shift in the source file (code inserted or deleted above the referenced
block), each range self-heals independently when you run `update` or
`check --heal`.

### When to use islands vs code blocks

Code blocks (`<!-- code -->`) wrap content in fenced code blocks with syntax
highlighting. They are the right choice for embedding source code examples.

Islands render content raw, with no wrapping. They are the right choice when
the source content is already formatted markdown that should flow naturally
into your README: prose, tables, lists, headings, or links.

### Use cases

#### Shared standards across repositories

Organisations often maintain coding standards, style guides, or architecture
decision records in a central repository. Teams reference these in their
project READMEs, but the references go stale as the standards evolve.

Islands let you pull the relevant sections directly. The content stays in sync
automatically, and cross-repo anchor links are rewritten so readers can click
through to the full document.

```markdown
<!-- island file="docs/python-style.md" repo="git@github.com:org/standards.git" ref="main" -->
<!-- lines from="12" to="30" -->
<!-- end island -->
```

This is particularly useful for tables of contents. If the standards repo has
a ToC with `[Installation](#12-installation)` links, those are rewritten to
`https://github.com/org/standards/blob/main/docs/python-style.md#12-installation`
so they resolve correctly from any consuming README.

#### Monorepo package documentation

In a monorepo, the root README often summarises each package. Rather than
duplicating the description from each package's own README (which drifts),
pull the introduction directly:

```markdown
## Packages

### Core

<!-- island file="packages/core/README.md" -->
<!-- lines from="3" to="12" -->
<!-- end island -->

### CLI

<!-- island file="packages/cli/README.md" -->
<!-- lines from="3" to="15" -->
<!-- end island -->
```

When a package author updates their README introduction, the root README
updates on the next `readme-merge update`.

#### Curated changelogs

A project changelog can grow long. The root README might want to show only the
latest release notes and a highlights section, without duplicating content that
already lives in CHANGELOG.md:

```markdown
## What's New

<!-- island file="CHANGELOG.md" -->
<!-- lines from="3" to="25" -->
<!-- end island -->

## Highlights

<!-- island file="CHANGELOG.md" -->
<!-- lines from="50" to="68" -->
<!-- end island -->
```

Multiple islands can reference different sections of the same file. Each range
is tracked independently, so if new entries push the highlights section down,
self-healing updates the line numbers.

#### API documentation from specs

If your API documentation lives in a separate spec file (OpenAPI description,
protocol docs, or a design document), you can surface key sections in the
README without maintaining a separate copy:

```markdown
## Authentication

<!-- island file="docs/api-spec.md" -->
<!-- lines from="45" to="78" -->
<!-- end island -->

## Rate Limits

<!-- island file="docs/api-spec.md" -->
<!-- lines from="112" to="130" -->
<!-- end island -->
```

#### Onboarding checklists from a wiki or runbook

Teams often maintain onboarding guides or runbooks separately from application
code. Pull the relevant setup steps into a CONTRIBUTING.md so new contributors
see current instructions:

```markdown
## Getting Started

<!-- island file="docs/onboarding.md" repo="git@github.com:org/wiki.git" ref="main" -->
<!-- lines from="15" to="45" -->
<!-- end island -->
```

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
readme-merge check [--source=<ref>] [--file=<path>] [--heal] [--full]
```

Exits 0 if everything is up to date. Exits 1 if any placeholder is stale or has
never been populated. By default, `check` is read-only and does not modify the
README. Pass `--heal` to write updated line references when snippets have shifted.

Each placeholder is listed with its source and a 3-line content preview. Pass
`--full` to show the entire content of each block.

Output examples:

```
$ readme-merge check
2 placeholder(s) fresh:
  src/client.go lines 10-15
    client := NewClient(os.Getenv("API_KEY"))
    resp, err := client.Send("hello")
    if err != nil {
    ... (3 more lines)
  git@github.com:org/shared-lib.git//README.md ref=main lines 1-5
    # shared-lib

    A shared library for...
    ... (2 more lines)

$ readme-merge check
1 stale placeholder(s):
  src/client.go lines 10-15
    client := NewClient(os.Getenv("API_KEY"))
    resp, err := client.Send("hello")
    if err != nil {
    ... (3 more lines)
1 placeholder(s) fresh:
  src/config.go lines 1-3
    package config

    var Version = "1.0.0"

$ readme-merge check --heal
self-healed 1 placeholder(s) (lines shifted)
1 placeholder(s) fresh:
  src/client.go lines 12-17
    client := NewClient(os.Getenv("API_KEY"))
    resp, err := client.Send("hello")
    if err != nil {
    ... (3 more lines)
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

### --full

Show the full content of each placeholder in `check` output. By default, only
the first 3 lines are shown with a count of remaining lines.

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

## Fence Escaping

When embedded content contains code fences (triple backticks), readme-merge
automatically uses longer fences (4+ backticks) for the outer wrapper. This
ensures nested code blocks render correctly on GitHub. For example, a markdown
file containing ` ```yaml ` blocks will be wrapped in ` ```` ` fences.

## Testing

Integration tests run the compiled binary against real files on disk, not
in-memory stubs. Each test scenario has its own directory under
`test/testdata/` containing a README and source files in their starting state:

```
test/testdata/
  basic/              # single code block
  selfheal/           # lines shift, snippet heals
  island-single/      # one island range
  island-multi/       # two island ranges
  island-many/        # four island ranges
  island-selfheal/    # island range heals after shift
  island-mixed/       # code block + island in same README
  ...
```

Tests call `readme-merge update` and `readme-merge check` as subprocesses, then
read the README back to assert on the result. After each test, `git checkout`
restores the fixture files to their original state.

This approach has two benefits. First, the test fixtures are reviewable: you can
open any `test/testdata/` directory and visually inspect the source files and
README to verify that the test scenario is correct. Second, the tests exercise
the real file I/O path, the real binary, and the real command-line interface,
not an abstraction that might diverge from production behaviour.

Unit tests (parser, engine, hasher, scanner) use in-memory strings and stub
readers for speed. The integration tests are the ones that prove the tool
actually works end to end.

## Security

Paths in local `from` attributes are validated to prevent directory traversal.
Absolute paths and paths that escape the project directory (e.g. `../secret.txt`)
are rejected. Only relative paths within the project root are allowed.

Cross-repo references use `git clone` and `git fetch` via SSH or HTTPS,
inheriting your existing git authentication (SSH agent, credential helpers).
No credentials are stored by readme-merge.

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
