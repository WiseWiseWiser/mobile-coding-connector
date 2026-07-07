# Scenario

**Bug**: empty git repo (no commits) must not abort dry-run; GIT REPOS shows error

```
# seed .wrk-test/empty (git init only) -> backup --dry-run -> GIT REPOS error line
remote-agent machine backup --dry-run -> exit 0; error: no commits (HEAD unborn)
```

## Preconditions

`git` on PATH; `SeedGitReposEmpty` runs `git init` under `.wrk-test/empty` with no add/commit.

## Steps

1. `SeedGitReposEmpty=true`.
2. Args: `machine backup --dry-run`.

## Context

REQUIREMENT leaf `backup/git-repos-empty-repo`; reproduces `.openclaw/workspace` HEAD unborn failure.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	requireGit(t)
	req.SeedGitReposEmpty = true
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```