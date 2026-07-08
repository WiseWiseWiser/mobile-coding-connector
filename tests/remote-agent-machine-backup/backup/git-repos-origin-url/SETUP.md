# Scenario

**Feature**: dry-run GIT REPOS and archive JSON include remote origin URL

```
# seed .wrk-test/main with origin remote -> dry-run + real backup -> origin line + origin_url JSON
SeedGitReposOrigin -> machine backup --dry-run then archive
```

## Preconditions

`git` on PATH; `SeedGitReposOrigin` seeds `.wrk-test/main` with commit message
`backup git fixture` and `origin` remote `https://github.com/example/backup-fixture.git`.

## Steps

1. `SeedGitReposOrigin=true`, `DryRunThenArchive=true`.
2. `OutputPath=git-repos-origin-url.tar.xz`.
3. Args: `machine backup --dry-run`.

## Context

REQUIREMENT leaf `backup/git-repos-origin-url`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	requireGit(t)
	req.SeedGitReposOrigin = true
	req.DryRunThenArchive = true
	req.OutputPath = "git-repos-origin-url.tar.xz"
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```