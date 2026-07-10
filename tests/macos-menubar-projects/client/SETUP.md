# Scenario

**Feature**: Swift source contracts for Projects menu and wrk API paths

```
AICriticApp.swift / ServerClient.swift -> Projects menu + /api/wrk/...
```

## Preconditions

`Op=client` inspects local macOS app sources under `macos-ai-critic/`.

## Steps

1. Set `Op=client`.
2. Leaf sets `ClientLeaf`.

## Context

REQUIREMENT optional scenarios 17–18.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "client"
	return nil
}
```
