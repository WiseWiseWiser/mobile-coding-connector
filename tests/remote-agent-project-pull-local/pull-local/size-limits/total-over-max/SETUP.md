# Scenario

**Feature**: default 64MB total package cap rejects large dirty sets

```
# 65 x 1 MiB untracked files + small diff over total max_size_bytes
remote-agent project pull-local -> server total cap -> exit 1
```

## Preconditions

Binding; bulk untracked files summing above 64 MiB, each file under per-file cap.

## Steps

1. `dirtyWithManyOneMiBFiles` with `bulkFileCountOver64` (65).
2. Default CLI max (64M); no `--max-size` override.

## Context

REQUIREMENT leaf `total-over-max`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	pair := pairSameOriginRepos(t)
	dirtyWithManyOneMiBFiles(t, pair.RemoteDir, bulkFileCountOver64)
	seedSizeLimitPullProject(t, req, "pull-size-total-001", "pull-size-total", pair)
	req.Args = []string{"project", "pull-local", "pull-size-total-001"}
	return nil
}
```