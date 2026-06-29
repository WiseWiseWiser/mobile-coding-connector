# Scenario

**Feature**: `--max-size 100M` allows package above default 64MB total

```
# same 65 MiB dirty set with raised max_size_bytes on server request
remote-agent project pull-local --max-size 100M -> exit 0, worktree has bulk files
```

## Preconditions

Same bulk layout as `total-over-max`; CLI passes `--max-size 100M`.

## Steps

1. `dirtyWithManyOneMiBFiles` with 65 chunks.
2. Args include `--max-size 100M`.

## Context

REQUIREMENT leaf `max-size-override`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	pair := pairSameOriginRepos(t)
	dirtyWithManyOneMiBFiles(t, pair.RemoteDir, bulkFileCountOver64)
	seedSizeLimitPullProject(t, req, "pull-size-max-001", "pull-size-max", pair)
	req.Args = []string{
		"project", "pull-local", "pull-size-max-001",
		"--max-size", "100M",
	}
	return nil
}
```