# Scenario

**Feature**: --git-dirs-scan-max-depth excludes repos beyond depth limit

```
# deep repo at .wrk-test/a/b/c/deep-repo with max-depth 2 -> GIT REPOS: (none)
```

## Preconditions

`git` on PATH; `SeedGitReposMaxDepth` nests repo deeper than scan cap.

## Steps

1. `SeedGitReposMaxDepth=true`, `GitDirsScanMaxDepth=2`.
2. Args: `machine backup --dry-run`.

## Context

REQUIREMENT leaf `backup/git-repos-max-depth`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	requireGit(t)
	req.SeedGitReposMaxDepth = true
	req.GitDirsScanMaxDepth = 2
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```