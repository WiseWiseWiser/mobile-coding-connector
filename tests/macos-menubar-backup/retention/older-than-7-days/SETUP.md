# Scenario

**Feature**: backups older than 7 calendar days are deleted

```
file 8 days before now -> delete
```

## Preconditions

now = 2026-07-10; file on 2026-07-02 is outside the 7-day past window
(window days: 07-03..07-09 plus today 07-10, depending on inclusive calendar rule:
**past days within 7 calendar days** of today → days with date in [today-7, today-1]
or [today-6, today-1]? Spec: "Past days within 7 calendar days" keep 1/day;
"Older / others" delete. Seal: a file on calendar day now.AddDate(0,0,-8) is deleted.

## Steps

1. One entry at 2026-07-02T12:00:00Z.

## Context

REQUIREMENT #19.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.NowRFC3339 = "2026-07-10T15:00:00Z"
	// 8 calendar days before 2026-07-10 is 2026-07-02
	req.EntriesJSON = `[
  {"path":"old-8d.tar.xz","mod_time":"2026-07-02T12:00:00Z","size_bytes":100}
]`
	return nil
}
```
