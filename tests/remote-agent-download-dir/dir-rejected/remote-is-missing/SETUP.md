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
import "testing"

func Setup(t *testing.T, req *Request) error {
	setDownloadArgs(t, req, "uploads/missing", "./local-missing")
	return nil
}
```