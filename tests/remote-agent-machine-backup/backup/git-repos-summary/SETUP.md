# Scenario

**Feature**: dry-run GIT REPOS section lists main repo metadata

```
# seed .wrk-test/main git repo -> backup --dry-run -> GIT REPOS summary block
remote-agent machine backup --dry-run -> branch, short sha, clean, commit msg
```

## Preconditions

`git` on PATH; `SeedGitRepos` seeds `.wrk-test/main` with commit message `backup git fixture`.

## Steps

1. `SeedGitRepos=true`.
2. Args: `machine backup --dry-run`.

## Context

REQUIREMENT leaf `backup/git-repos-summary`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	requireGit(t)
	req.SeedGitRepos = true
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```