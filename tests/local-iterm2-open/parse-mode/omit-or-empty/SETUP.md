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
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ModeInput = ""
	return nil
}
```
