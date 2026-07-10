# Scenario

**Feature**: ServerClient applies Authorization Bearer when token present

```
ServerClient.swift (+ LocalAuth helpers) -> Authorization Bearer on requests
# empty token must not force bare "Bearer "
```

## Preconditions

Local product sources exist under `macos-ai-critic/ai-critic-macos/` (and/or
Shared). RED until Authorization Bearer is wired into the request path.

## Steps

1. Set `ClientLeaf=serverclient-sets-bearer`.

## Context

REQUIREMENT leaf: scenario 8.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "serverclient-sets-bearer"
	return nil
}
```
