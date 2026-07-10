# Scenario

**Feature**: mixed today + history + old applies all three retention rules

```
today(3) + yesterday(2) + day-3(1) + day-8(1) -> keep today all 3 + y newest + d3 + delete rest
```

## Preconditions

Fewer than 10 today so all today kept; yesterday keeps one; 8d-old deleted.

## Steps

1. Compose EntriesJSON with seven files across days.

## Context

REQUIREMENT #20.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.NowRFC3339 = "2026-07-10T15:00:00Z"
	req.EntriesJSON = `[
  {"path":"t1.tar.xz","mod_time":"2026-07-10T10:00:00Z","size_bytes":1},
  {"path":"t2.tar.xz","mod_time":"2026-07-10T12:00:00Z","size_bytes":2},
  {"path":"t3.tar.xz","mod_time":"2026-07-10T14:00:00Z","size_bytes":3},
  {"path":"y1.tar.xz","mod_time":"2026-07-09T09:00:00Z","size_bytes":4},
  {"path":"y2.tar.xz","mod_time":"2026-07-09T20:00:00Z","size_bytes":5},
  {"path":"d3.tar.xz","mod_time":"2026-07-07T12:00:00Z","size_bytes":6},
  {"path":"old.tar.xz","mod_time":"2026-07-02T12:00:00Z","size_bytes":7}
]`
	return nil
}
```
