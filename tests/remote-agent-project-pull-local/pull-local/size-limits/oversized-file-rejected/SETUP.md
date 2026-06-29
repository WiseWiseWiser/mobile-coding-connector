# Scenario

**Feature**: per-file 1MB cap rejects oversized untracked without include

```
# 2MB big.bin in dirty set, server dry_run guard
remote-agent project pull-local -> POST pull-local -> 400 / CLI exit 1
```

## Preconditions

Seeded binding; remote dirty with `big.bin` at 2 MiB.

## Steps

1. `pairSameOriginRepos`; `dirtyWithBigUntracked`.
2. Project `pull-size-big-001`; argv without `--include-file`.

## Context

REQUIREMENT leaf `oversized-file-rejected`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	pair := pairSameOriginRepos(t)
	dirtyWithBigUntracked(t, pair.RemoteDir)
	seedSizeLimitPullProject(t, req, "pull-size-big-001", "pull-size-big", pair)
	req.Args = []string{"project", "pull-local", "pull-size-big-001"}
	return nil
}
```