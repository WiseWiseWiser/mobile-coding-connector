# Consolidate git/worktree Overlap (PR-6)

## Investigation summary

After PR-5, remaining duplication lived in `git/worktree/` vs `git/scan_repo/` and
wrkcli clean checks.

| Concern | Duplicate locations | Fix |
|---------|-------------------|-----|
| `git worktree list --porcelain` parse | `worktree.parsePorcelain`, `scan_repo.parseWorktrees` | Export `ParseListPorcelain`; `scan_repo` calls `worktree.ListCtx` |
| Read-only git subprocess | `worktree.ReadBranch`, `worktree.List`, `worktree.IsClean`, `merge_back.gitOutput` | Route through `git/cmd` |
| Wrk is-clean check | `wrkcli.gitWorktreeIsClean` (inline ParsePorcelainWrk) | `worktree.IsCleanWrk` |

**Not consolidated (intentional):**
- `git/status` backup vs wrk taxonomies — different format strings
- `wrkcli/gitexec.go` mutating git (`fetch`, `worktree add`) — keeps `exec.Cmd` + verbose logging
- `worktree.IsInsideWorkTree` — still uses raw exec (gitops path); low churn

## Implemented API

```go
// git/worktree
func ParseListPorcelain(output string) []Entry
func ListCtx(ctx context.Context, repoPath string) ([]Entry, error)
func ReadBranchCtx(ctx context.Context, worktreePath string) (string, error)
func IsCleanWrk(path string) (bool, error)
```

`scan_repo.listWorktrees` → `worktree.ListCtx` + `scanWorktreesFromEntries`.

## Verification

All enrich-worktrees, wrk status, and worktree merge-back doctests pass.