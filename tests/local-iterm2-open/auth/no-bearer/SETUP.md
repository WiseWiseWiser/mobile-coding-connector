# Scenario

**Feature**: missing Authorization yields 401

```
POST open without Bearer -> 401 {"error":"unauthorized"|...}
```

## Preconditions

Credentials file present (server initialized).

## Steps

1. Set `OmitAuth=true`.

## Context

REQUIREMENT: without auth → 401.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.OmitAuth = true
	req.BearerToken = ""
	return nil
}
```
