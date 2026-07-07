# Scenario

**Feature**: --skip-git-dirs-scan skips git scan in summary and archive

```
# git fixtures present but --skip-git-dirs-scan -> GIT REPOS: (skipped); no archive JSON
dry-run then real backup via DryRunThenArchive
```

## Preconditions

`git` on PATH; `SeedGitRepos` seeds a repo but scan is skipped via flag.

## Steps

1. `SeedGitRepos=true`, `SkipGitDirsScan=true`, `DryRunThenArchive=true`.
2. `OutputPath=git-repos-skipped.tar.xz`.
3. Args: `machine backup --dry-run`.

## Context

REQUIREMENT leaf `backup/git-repos-skipped`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	requireGit(t)
	req.SeedGitRepos = true
	req.SkipGitDirsScan = true
	req.DryRunThenArchive = true
	req.OutputPath = "git-repos-skipped.tar.xz"
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```