# Scenario

**Feature**: multiple backups on a past day keep only the newest

```
3 files yesterday -> keep newest only for that day
```

## Preconditions

Yesterday is within the 7-day window; no today files needed.

## Steps

1. Three entries on 2026-07-09 at 10:00, 12:00, 18:00 UTC.

## Context

REQUIREMENT #18.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.NowRFC3339 = "2026-07-10T15:00:00Z"
	req.EntriesJSON = `[
  {"path":"y-early.tar.xz","mod_time":"2026-07-09T10:00:00Z","size_bytes":1},
  {"path":"y-mid.tar.xz","mod_time":"2026-07-09T12:00:00Z","size_bytes":2},
  {"path":"y-late.tar.xz","mod_time":"2026-07-09T18:00:00Z","size_bytes":3}
]`
	return nil
}
```
