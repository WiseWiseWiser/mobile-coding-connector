# Scenario

**Feature**: HOME-wide git scan discovers repo at non-dot top-level path

```
# seed ~/projects/demo git repo -> backup --dry-run -> GIT REPOS lists projects/demo
remote-agent machine backup --dry-run -> branch, short sha, clean, commit msg
```

## Preconditions

`git` on PATH; `SeedGitReposNonDot` seeds `projects/demo` with commit message `non-dot fixture`.

## Steps

1. `SeedGitReposNonDot=true`.
2. Args: `machine backup --dry-run`.

## Context

REQUIREMENT leaf `backup/git-repos-non-dot-path` (HOME-wide git scan).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	requireGit(t)
	req.SeedGitReposNonDot = true
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```