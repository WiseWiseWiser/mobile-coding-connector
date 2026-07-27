# Scenario

**Feature**: restore --dry-run skips byte-identical paths

```
# backup then restore --dry-run via /restore/stream with unchanged serverHome
CLASSIFYING: skip (identical) shortcut; no APPLYING; dry-run: machine restore plan; no writes
```

## Preconditions

Prereq backup; no post-backup mutation.

## Steps

1. `AfterBackupMutate` empty.
2. Args: `machine restore --dry-run` (archive injected by Run).

## Context

REQUIREMENT leaf `restore/dry-run-identical`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.AfterBackupMutate = ""
	req.Args = []string{"machine", "restore", "--dry-run"}
	return nil
}
```