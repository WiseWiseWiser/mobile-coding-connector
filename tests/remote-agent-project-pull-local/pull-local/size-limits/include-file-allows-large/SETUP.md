# Scenario

**Feature**: `--include-file` exempts a single oversized dirty path

```
# 2MB big.bin listed in include_files on server request
remote-agent project pull-local --include-file big.bin -> package -> worktree
```

## Preconditions

Same dirty layout as oversized rejection; include lists `big.bin`.

## Steps

1. `dirtyWithBigUntracked` on shared-origin pair.
2. Args include `--include-file big.bin`.

## Context

REQUIREMENT leaf `include-file-allows-large`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	pair := pairSameOriginRepos(t)
	dirtyWithBigUntracked(t, pair.RemoteDir)
	seedSizeLimitPullProject(t, req, "pull-size-inc-001", "pull-size-inc", pair)
	req.Args = []string{
		"project", "pull-local", "pull-size-inc-001",
		"--include-file", bigUntrackedRel,
	}
	return nil
}
```