# Scenario

**Feature**: guided connection status copy (no raw token)

```
ConnectionState + server -> remoteconfig.FormatStatus -> user-facing status line
```

## Preconditions

Status strings are stable product copy. Never include the token. States that
need user action mention Configure… where specified.

## Steps

1. Set `Op=status`.
2. Leaf sets `ConnectionState` and optional `StatusServer` / sentinel `Token`.

## Context

REQUIREMENT group: `status/`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "status"
	return nil
}
```
