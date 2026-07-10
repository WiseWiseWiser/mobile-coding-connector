# Scenario

**Feature**: SSE section frame line

```
FormatBackupProgressSection("Collecting files") -> "[section] Collecting files"
```

## Preconditions

SSE `type=section` with `message`.

## Steps

1. Op=format_section; Message set.

## Context

REQUIREMENT #9.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "format_section"
	req.Message = "Collecting files"
	return nil
}
```
