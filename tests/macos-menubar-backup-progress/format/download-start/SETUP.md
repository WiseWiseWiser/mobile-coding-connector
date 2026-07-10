# Scenario

**Feature**: local download phase start line

```
FormatBackupProgressDownloadStart() -> "Downloading archive…"
```

## Preconditions

After stream `done` with token, before archive GET completes.

## Steps

1. Op=format_download_start.

## Context

REQUIREMENT format table download start (ellipsis `…` U+2026).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "format_download_start"
	return nil
}
```
