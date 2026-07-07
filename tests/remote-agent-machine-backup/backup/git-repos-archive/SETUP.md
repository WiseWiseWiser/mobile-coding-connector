# Scenario

**Feature**: real backup archive embeds git-repo-worktrees.json meta

```
# seed .wrk-test/main -> machine backup --output -> archive contains git JSON snapshot
```

## Preconditions

`git` on PATH; `SeedGitRepos` seeds included dot-dir git fixture.

## Steps

1. `SeedGitRepos=true`.
2. `OutputPath=git-repos-archive.tar.xz`.
3. Args: `machine backup --output __OUTPUT_PATH__`.

## Context

REQUIREMENT leaf `backup/git-repos-archive`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	requireGit(t)
	req.SeedGitRepos = true
	req.OutputPath = "git-repos-archive.tar.xz"
	req.Args = []string{"machine", "backup", "--output", "__OUTPUT_PATH__"}
	return nil
}
```