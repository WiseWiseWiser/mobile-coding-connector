# Scenario

**Feature**: Register mounts POST /api/local/iterm2/open

```
Register(mux) -> POST /api/local/iterm2/open -> not 404
```

## Preconditions

Valid temp dir for body.

## Steps

1. Set `Op=register`.
2. `OmitMode=true`; Dir filled by Run if empty.

## Context

REQUIREMENT: mount on server mux.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "register"
	req.Dir = t.TempDir()
	req.OmitMode = true
	req.OmitSend = true
	return nil
}
```
