# Scenario

**Feature**: success write line with path and human size

```
FormatBackupProgressWrote(path, 42MiB) -> "Wrote {path} (42 MB)"
```

## Preconditions

Size is exact `42 * 1024 * 1024` bytes; same unit style as recent list (`MB`).

## Steps

1. Op=format_wrote; fixed path and size.

## Context

REQUIREMENT #12.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "format_wrote"
	req.Path = "/Users/u/.backup/ai-critic/foo.example.com/machine-backup-20260710-150000.tar.xz"
	req.SizeBytes = 42 * 1024 * 1024
	return nil
}
```
