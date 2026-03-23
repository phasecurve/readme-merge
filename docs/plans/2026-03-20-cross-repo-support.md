# Cross-Repository Support for readme-merge

## Problem Statement

readme-merge can only reference files within the current repository. Teams that maintain shared libraries (e.g. shared-lib) want consuming repositories (e.g. consuming-service) to embed documentation snippets from the shared repo's README. Today this requires manual copy-paste, which defeats the purpose of readme-merge.

## Scope

### In Scope

- Extend the `from=` directive to accept git remote URLs with an in-repo path
- Add a new `ref=` attribute for pinning the git ref on cross-repo references
- Clone remote repos into a local cache for content extraction
- Support both SSH and HTTPS URLs (whichever git handles natively)
- Staleness detection and self-healing for cross-repo blocks
- Cache management (creation, reuse, freshening)

### Out of Scope

- Authentication configuration (relies on git's own SSH agent / credential helpers)
- Cache garbage collection CLI command (can be added later)
- Submodule-based approaches
- Monorepo path remapping

## Requirements

### Functional Requirements

| ID | Requirement | Acceptance Criteria |
|----|-------------|---------------------|
| FR-1 | Parse cross-repo `from=` values containing a git URL and `//`-separated file path, plus a separate `ref=` attribute | `from=git@github.com:org/repo.git//README.md ref=main lines=30-45` is parsed into repo URL `git@github.com:org/repo.git`, file path `README.md`, ref `main` |
| FR-2 | When `ref=` is omitted on a cross-repo block, default to `main` | `from=git@github.com:org/repo.git//README.md lines=30-45` is treated as ref `main` |
| FR-3 | Clone the remote repo into `.readme-merge/cache/<hash>/` on first access | Running `update` with a cross-repo directive creates the cache directory and clones the repo |
| FR-4 | Reuse cached clone on subsequent runs, fetching the latest ref | Second `update` run does `git fetch` + `git checkout` in the cached clone, not a full re-clone |
| FR-5 | Extract lines from the cached file using the same engine logic as local files | Cross-repo blocks produce identical output format (code fence, hashes) as local blocks |
| FR-6 | Staleness detection works for cross-repo blocks | `check` compares hashes against the remote file content at the pinned ref |
| FR-7 | Self-healing works for cross-repo blocks | When lines shift in the remote file, the snippet hash scanner finds the new location |
| FR-8 | Local `from=` paths continue to work unchanged | All existing tests pass without modification |
| FR-9 | HTTPS URLs work if git handles them | `from=https://github.com/org/repo.git//file.py ref=v1.0 lines=1-10` works when git credentials are available |
| FR-10 | The `--source` flag applies only to local blocks, not cross-repo blocks | Cross-repo blocks always read from their pinned ref, ignoring `--source=staged` etc. |
| FR-11 | `ref=` attribute is ignored on local blocks | `from=src/main.go ref=main lines=1-10` ignores `ref=` and reads from the local resolver as normal |

### Non-Functional Requirements

| ID | Requirement | Acceptance Criteria |
|----|-------------|---------------------|
| NFR-1 | No new external Go dependencies | Implementation uses only the standard library and git CLI |
| NFR-2 | Cache directory is deterministic from the repo URL | Same URL always maps to the same cache directory |
| NFR-3 | First clone of a repo completes in under 30 seconds for repos under 100MB | Shallow clone (`--depth=1`) is used |
| NFR-4 | `.readme-merge/` is added to the project's `.gitignore` suggestion in output | `update` prints a warning if `.readme-merge/` is not in `.gitignore` |

## Behaviours

### Placeholder syntax (cross-repo)

```markdown
<!-- code from=git@github.com:org/repo.git//path/to/file.md ref=branch-or-tag lines=30-45 -->
<!-- /code -->
```

| Attribute | Required | Description |
|-----------|----------|-------------|
| `from`    | yes      | For cross-repo: `<git-url>//<file-path>`. The `//` separates the repo URL from the in-repo file path. For local: relative path (unchanged) |
| `ref`     | no       | Git ref (branch, tag, SHA) for cross-repo blocks. Defaults to `main` if omitted. Ignored on local blocks. |
| `lines`   | yes      | Line range to extract, inclusive (e.g. `30-45`) |

Auto-populated after `update` (same as local blocks):

| Attribute      | Description |
|----------------|-------------|
| `filehash`     | SHA-256 hash of the entire source file (truncated to 16 hex chars) |
| `snippethash`  | SHA-256 hash of just the extracted lines |

### Parsing a cross-repo `from=` value

**Trigger**: Parser encounters a `from=` value containing `//`

**Detection rule**: If the `from=` value contains `//`, it is a cross-repo reference. Everything before `//` is the repo URL. Everything after `//` is the file path within that repo.

**Input**: `from=git@github.com:org/shared-lib.git//README.md ref=chore/clean-docs lines=30-45`

**Parse result**:
- Repo URL: `git@github.com:org/shared-lib.git`
- File path: `README.md`
- Ref: `chore/clean-docs` (from `ref=` attribute)
- Lines: 30-45

**Without ref**: `from=git@github.com:org/shared-lib.git//README.md lines=30-45`

**Parse result**:
- Repo URL: `git@github.com:org/shared-lib.git`
- File path: `README.md`
- Ref: `main` (default)
- Lines: 30-45

**Errors**:
- `from=` contains `//` but no file path after it: `"cross-repo reference missing file path: <from>"`
- File path after `//` is empty or whitespace: same error

### Parser regex changes

The existing `openRe` regex captures `from=(\S+)`. This already accepts URLs with `:`, `@`, `/`.

Add an optional `ref=` capture group to the regex. The attribute order in the directive is: `from=`, then optionally `ref=`, then `lines=`, then optionally `filehash=` and `snippethash=`. The `ref=` attribute is new.

Updated regex pattern:
```
<!--\s*code\s+from=(\S+)\s+(?:ref=(\S+)\s+)?lines=(\d+)-(\d+)(?:\s+filehash=(\S+))?(?:\s+snippethash=(\S+))?\s*-->
```

Capture groups:
1. `from` value (repo URL + `//` + path, or local path)
2. `ref` value (optional, nil if absent)
3. Line start
4. Line end
5. File hash (optional)
6. Snippet hash (optional)

### Block struct changes

Add `Ref` field to `parser.Block`:

```go
type Block struct {
    From        string
    Ref         string    // git ref for cross-repo blocks; empty for local
    SourceStart int
    SourceEnd   int
    FileHash    string
    SnippetHash string
    Content     string
    ReadmeStart int
    ReadmeEnd   int
}
```

### Resolving a cross-repo file

**Trigger**: Engine encounters a block where `From` contains `//`

**Process**:

1. Split `From` on `//`: repo URL = before, file path = after
2. Determine ref: use `Block.Ref` if non-empty, otherwise `main`
3. Compute cache directory: `.readme-merge/cache/<sha256_hex_16(repo_url)>/`
4. If cache directory does not exist:
   a. Run `git clone --depth=1 --branch=<ref> --single-branch <repo_url> <cache_dir>`
   b. If clone fails, return error: `"cloning <repo_url>: <git stderr>"`
5. If cache directory exists:
   a. Run `git -C <cache_dir> fetch origin <ref>`
   b. Run `git -C <cache_dir> checkout FETCH_HEAD`
   c. If fetch fails, return error: `"fetching <ref> from <repo_url>: <git stderr>"`
6. Read `<cache_dir>/<file_path>` from disk
7. Return file content to engine

**Output**: File content string, identical to what `source.Resolver.ReadFile` returns for local files

**Errors**:
- Git clone/fetch failure (network, auth, ref not found): `"cloning <url>: <message>"` or `"fetching <ref> from <url>: <message>"`
- File not found in cloned repo: `"reading <path> in <url> ref=<ref>: file not found"`

### Cache directory naming

**Input**: Repo URL string (e.g. `git@github.com:org/shared-lib.git`)

**Output**: SHA-256 of the URL, truncated to 16 hex chars (same scheme as existing `hasher.ContentHash`)

**Rationale**: Avoids filesystem-unsafe characters from URLs. Deterministic, so the same repo always maps to the same directory.

### Rendering cross-repo blocks

**Behaviour**: Identical to local blocks, plus the `ref=` attribute is preserved in the rendered HTML comment.

**Example output**:
```markdown
<!-- code from=git@github.com:org/shared-lib.git//README.md ref=main lines=30-45 filehash=abc123... snippethash=def456... -->
```markdown
content here
```
<!-- /code -->
```

The `Render` function includes `ref=` in the output only when `Block.Ref` is non-empty.

### Interaction with `--source` flag

**Behaviour**: The `--source` flag (staged, HEAD, git ref) applies only to local file resolution. Cross-repo blocks always resolve against their own `ref=` in the cached clone. This is because `--source=staged` means "the staged version of files in this repo", which has no meaning for a remote repository.

## Edge Cases

| Case | Expected Behaviour |
|------|-------------------|
| `from=` contains `//` but is a local path like `docs//file.md` | Treated as cross-repo. Fails at clone step with a clear error. `//` in local paths is abnormal and unsupported. |
| `ref=` contains `/` (e.g. `ref=chore/clean-docs`) | Works. The regex captures `\S+` which allows `/`. |
| `ref=` specified on a local block | Ignored. Local blocks use the local resolver regardless. |
| Network unavailable during `update` | Error: `"cloning <url>: <git stderr>"`. The command fails, README is not modified. |
| Cache exists but ref no longer exists on remote | Error: `"fetching <ref> from <url>: <git stderr>"` |
| Multiple blocks reference the same remote repo with same ref | The repo is cloned/fetched once. The resolver caches resolved repos in memory for the duration of the run. |
| Multiple blocks reference the same remote repo with different refs | Each unique (repo, ref) pair is fetched separately. Since there is one cache directory per repo, the checkout switches between refs. Blocks are processed sequentially, so this works correctly. |
| Mixed local and cross-repo blocks in one README | Both work. Local blocks use the local resolver, cross-repo blocks use the remote resolver. |

## Error Handling

| Error Condition | Detection | Response | Recovery |
|-----------------|-----------|----------|----------|
| Git not installed | `exec.LookPath("git")` fails | `"git is required for cross-repo references"` | User installs git |
| SSH key not available | Git clone fails with auth error | Pass through git's stderr: `"cloning <url>: Permission denied (publickey)"` | User configures SSH agent |
| Ref not found | Git fetch/clone fails | `"fetching <ref> from <url>: <git stderr>"` | User fixes ref name |
| File not in repo | `os.ReadFile` after checkout fails | `"reading <path> in <url> ref=<ref>: file not found"` | User fixes file path |
| Line range out of bounds | Same as local blocks | `"block <from>: line range X-Y out of bounds (Z lines)"` | User fixes line range |
| Cache directory not writable | `os.MkdirAll` fails | `"creating cache directory: <error>"` | User fixes permissions |
| Network timeout | Git operation hangs | Git's own timeout behaviour (no custom timeout) | User retries |

## Dependencies

### External Systems
- Git CLI (already a dependency for `--source` flag). Cross-repo extends this to cloning/fetching.

### Libraries/Packages
- None new. Uses `os/exec`, `crypto/sha256`, `path/filepath` from standard library.

## Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| Cache directory | Implicit | `.readme-merge/cache/` relative to project root | Where cloned repos are stored |
| Default ref | Implicit | `main` | Used when `ref=` is omitted from cross-repo blocks |

No new CLI flags are introduced. The feature is driven entirely by the directive syntax.

## Implementation Notes

### Architecture fit

The engine already uses the `FileReader` interface:

```go
type FileReader interface {
    ReadFile(path string) (string, error)
}
```

Cross-repo support requires a composite resolver that:
1. Inspects the `from=` value
2. If it contains `//`, delegates to a `RemoteResolver` that handles clone/cache/read
3. Otherwise, delegates to the existing `source.Resolver`

The parser's `Block.From` field stores the full `from=` value unchanged. The `Block.Ref` field stores the parsed `ref=` value. The engine passes both to the composite resolver.

### New package: `internal/remote/`

- `remote.go`: `RemoteResolver` struct
  - `Resolve(fromValue, ref string) (string, error)` - parses the from= value, manages cache, returns file content
  - `parseFromValue(from string) (repoURL, filePath string, err error)` - splits on `//`
  - `ensureClone(repoURL, ref, cacheDir string) error` - clone or fetch+checkout
  - `cacheDir(repoURL string) string` - deterministic cache path
- `remote_test.go`: Unit tests for URL parsing, cache path computation

### Changes to existing code

- `parser.go`: Update `openRe` regex to capture optional `ref=` group. Update `Block` struct with `Ref` field. Update `Parse` to populate `Ref`. Update `Render` to include `ref=` when non-empty.
- `engine.go`: The `FileReader` interface method signature changes to accept both `from` and `ref` values, or the engine pre-processes blocks to route to the correct reader. Simplest approach: engine checks for `//` in `Block.From` and calls the appropriate reader.
- `main.go`: Create composite resolver wrapping both local and remote, pass project root for cache directory location.
- `source.go`: No change needed (only called for local paths).

### Files to create

| File | Purpose |
|------|---------|
| `internal/remote/remote.go` | RemoteResolver: parse cross-repo from=, manage cache, read files |
| `internal/remote/remote_test.go` | Unit tests for URL parsing, cache path computation |

### Files to modify

| File | Change |
|------|--------|
| `internal/parser/parser.go` | Add `ref=` to regex, add `Ref` to Block, update Parse and Render |
| `internal/engine/engine.go` | Route cross-repo blocks to RemoteResolver |
| `cmd/readme-merge/main.go` | Create composite resolver, pass project root |
| `README.md` | Document cross-repo syntax and `ref=` attribute |
| `test/integration_test.go` | Add cross-repo integration test |

## Acceptance Criteria Summary

- [ ] FR-1: `from=git@github.com:org/repo.git//path ref=main lines=N-M` is correctly parsed
- [ ] FR-2: Omitted `ref=` defaults to `main` for cross-repo blocks
- [ ] FR-3: First run clones into `.readme-merge/cache/<hash>/`
- [ ] FR-4: Subsequent runs fetch+checkout instead of re-cloning
- [ ] FR-5: Cross-repo blocks produce identical output format as local blocks
- [ ] FR-6: `check` detects stale cross-repo blocks
- [ ] FR-7: Self-healing updates line numbers when remote content shifts
- [ ] FR-8: All existing local-only tests pass unchanged
- [ ] FR-9: HTTPS URLs work when git credentials are available
- [ ] FR-10: `--source=staged` does not affect cross-repo blocks
- [ ] FR-11: `ref=` on local blocks is silently ignored
- [ ] NFR-1: No new Go dependencies added to go.mod
- [ ] NFR-2: Same repo URL always maps to same cache directory
- [ ] NFR-3: Clone uses `--depth=1` for speed
- [ ] NFR-4: Warning printed if `.readme-merge/` not in `.gitignore`

## Open Questions

None. All behavioural decisions have been resolved.

## Decisions Log

| Decision | Rationale | Date |
|----------|-----------|------|
| Syntax: `from=url//path` with separate `ref=` attribute | Clean separation of concerns; `from=` is the locator, `ref=` is the version. Avoids `@`-in-file-paths parsing ambiguity entirely. | 2026-03-20 |
| Default ref is `main` when `ref=` omitted | Pragmatic default for most workflows | 2026-03-20 |
| Rely on git CLI for auth (SSH agent, credential helpers) | Avoids reinventing credential management, works with existing user setup | 2026-03-20 |
| Cache in `.readme-merge/cache/` in project root | Project-local, easy to gitignore, no global state | 2026-03-20 |
| Shallow clone (`--depth=1`) | Speed and disk usage; we only need the content at one ref | 2026-03-20 |
| `--source` flag ignored for cross-repo blocks | `--source=staged` has no meaning for remote repos | 2026-03-20 |
| `ref=` ignored on local blocks | Keeps local behaviour unchanged, no breaking changes | 2026-03-20 |
