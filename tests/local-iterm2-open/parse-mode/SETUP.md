# Scenario

**Feature**: ParseOpenMode maps JSON mode strings to iterm2.OpenMode

```
mode string -> ParseOpenMode -> ModeReuseCurrent | ModeForceNew | ModeSmart | error
```

## Preconditions

Pure function; no filesystem or HTTP.

## Steps

1. Set `Op=parse`.
2. Leaf sets `ModeInput`.

## Context

REQUIREMENT mode mapping table; empty/omit → reuse (not lib zero ModeSmart).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "parse"
	return nil
}
```
