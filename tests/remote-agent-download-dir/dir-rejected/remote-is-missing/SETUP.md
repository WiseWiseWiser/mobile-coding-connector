# Scenario

**Feature**: download fails when remote path is missing

```
# uploads/missing absent on server -> download -> error, no local files
missing remote path -> remote-agent download -> non-zero exit
```

## Preconditions

`uploads/missing` does not exist under `serverHome`.

## Steps

1. Args: `download uploads/missing ./local-missing`.

## Context

REQUIREMENT leaf #7 — dir-rejected/remote-is-missing.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	// L3 smoke: missing remote reject via product binaries.
	req.UseCLI = true
	setDownloadArgs(t, req, "uploads/missing", "./local-missing")
	return nil
}
```