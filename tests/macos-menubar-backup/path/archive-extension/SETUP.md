# Scenario

**Feature**: archive filename uses UTC timestamp and `.tar.xz`

```
BackupArchiveFilename(2026-07-10T12:00:00Z) -> "machine-backup-20260710-120000.tar.xz"
```

## Preconditions

Filename matches CLI machine backup naming: `machine-backup-<YYYYMMDD-HHMMSS>.tar.xz` in UTC.

## Steps

1. Set `Op=path_archive_filename`, UTC `2026-07-10T12:00:00Z`.

## Context

REQUIREMENT leaf: paths #10; CLI uses same pattern.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "path_archive_filename"
	req.UTCTimeRFC3339 = "2026-07-10T12:00:00Z"
	return nil
}
```
