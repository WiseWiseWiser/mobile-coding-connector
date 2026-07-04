# Scenario

**Feature**: both conflicts cleared by `--kill-existing`

```
kill server occupier + daemon occupier -> daemon binds both roles -> running
```

## Preconditions

Parent `both-occupied` setup.

## Steps

1. `StartupWaitSecs=20`.

## Context

REQUIREMENT leaf: `both-occupied/kills-both`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.StartupWaitSecs = 20
	return nil
}
```