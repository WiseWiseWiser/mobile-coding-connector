# Scenario

**Feature**: recent row formats relative time and human size

```
FormatBackupEntry(mod=now-12m, size=42MiB, now) -> "12m ago · 42 MB"
```

## Preconditions

Size is exact 42 * 1024 * 1024 bytes; modTime 12 minutes before now.

## Steps

1. Set `Op=recent_format` with fixed now and entry fields.

## Context

REQUIREMENT #16.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "recent_format"
	req.NowRFC3339 = "2026-07-10T15:00:00Z"
	req.EntryPath = "machine-backup-20260710-144800.tar.xz"
	req.EntryModRFC3339 = "2026-07-10T14:48:00Z" // 12m ago
	req.EntrySizeBytes = 42 * 1024 * 1024
	return nil
}
```
