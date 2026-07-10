# Scenario

**Feature**: recent entries ordered newest first by modTime

```
SortBackupEntriesNewestFirst([old, mid, new]) -> [new, mid, old]
```

## Preconditions

Three synthetic entries with distinct mod times.

## Steps

1. Set `Op=recent_list` with JSON entries a/b/c.

## Context

REQUIREMENT #15.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "recent_list"
	req.EntriesJSON = `[
  {"path":"old.tar.xz","mod_time":"2026-07-10T10:00:00Z","size_bytes":1},
  {"path":"new.tar.xz","mod_time":"2026-07-10T14:00:00Z","size_bytes":3},
  {"path":"mid.tar.xz","mod_time":"2026-07-10T12:00:00Z","size_bytes":2}
]`
	return nil
}
```
