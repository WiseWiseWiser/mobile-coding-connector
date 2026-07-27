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
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "format_start_header"
	req.ServerName = "foo.example.com"
	return nil
}
```
