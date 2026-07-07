# Scenario

**Feature**: backup --exclude does not affect HOME-wide git scan

```
# seed .wrk-test/main + --exclude .wrk-test/** -> GIT REPOS still lists .wrk-test/main
remote-agent machine backup --dry-run --exclude .wrk-test/** -> git scan independent of exclusions
```

## Preconditions

`git` on PATH; `SeedGitRepos` seeds `.wrk-test/main`. `--exclude .wrk-test/**` excludes
the tree from backup plan/archive but must not suppress git repo discovery.

## Steps

1. `SeedGitRepos=true`, `ExcludePaths=[".wrk-test/**"]`.
2. Args: `machine backup --dry-run`.

## Context

REQUIREMENT leaf `backup/git-repos-exclude-independent` (exclude-independent git scan).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	requireGit(t)
	req.SeedGitRepos = true
	req.ExcludePaths = []string{".wrk-test/**"}
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```