# Scenario

**Feature**: dry-run GIT REPOS reports origin (none) when repo has no remotes

```
# default SeedGitRepos fixture without remote -> backup --dry-run -> origin (none)
```

## Preconditions

`git` on PATH; `SeedGitRepos` seeds `.wrk-test/main` with no `origin` remote.

## Steps

1. `SeedGitRepos=true`.
2. Args: `machine backup --dry-run`.

## Context

REQUIREMENT leaf `backup/git-repos-no-origin`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	requireGit(t)
	req.SeedGitRepos = true
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```