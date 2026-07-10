# Scenario

**Feature**: History is a disabled placeholder

```
Button("History...") .disabled(true)  // no history API in menu yet
```

## Preconditions

Swift sources for local and/or remote menu-bar apps are present.

## Steps

1. Set `ClientLeaf=history-disabled`.

## Context

REQUIREMENT leaf: `client/history-disabled`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "history-disabled"
	return nil
}
```
