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
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "register"
	req.Dir = t.TempDir()
	req.OmitMode = true
	req.OmitSend = true
	return nil
}
```
