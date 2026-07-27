# Scenario

**Feature**: empty mode string defaults to reuse

```
ParseOpenMode("") -> ModeReuseCurrent
```

## Preconditions

Handler will pass empty when JSON mode is omitted or empty.

## Steps

1. Set `ModeInput=""`.

## Context

REQUIREMENT: empty / omit → reuse (`ModeReuseCurrent`).

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ModeInput = ""
	return nil
}
```
