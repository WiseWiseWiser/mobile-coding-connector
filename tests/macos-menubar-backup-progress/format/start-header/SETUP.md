# Scenario

**Feature**: progress header includes server name

```
FormatBackupProgressStartHeader("foo.example.com") -> "Machine backup — foo.example.com"
```

## Preconditions

Server scope key is the display name after URL host extraction.

## Steps

1. Op=format_start_header; ServerName=foo.example.com.

## Context

REQUIREMENT #8.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "format_start_header"
	req.ServerName = "foo.example.com"
	return nil
}
```
