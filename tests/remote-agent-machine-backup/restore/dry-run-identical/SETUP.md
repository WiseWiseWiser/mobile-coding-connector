# Scenario

**Feature**: restore --dry-run skips byte-identical paths

```
# backup then restore --dry-run via /restore/stream with unchanged serverHome
stream skip (identical) lines + dry-run: machine restore plan summary; no writes
```

## Preconditions

Prereq backup; no post-backup mutation.

## Steps

1. `AfterBackupMutate` empty.
2. Args: `machine restore --dry-run` (archive injected by Run).

## Context

REQUIREMENT leaf `restore/dry-run-identical`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.AfterBackupMutate = ""
	req.Args = []string{"machine", "restore", "--dry-run"}
	return nil
}
```