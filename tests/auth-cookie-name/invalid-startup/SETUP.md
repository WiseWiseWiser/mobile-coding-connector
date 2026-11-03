# Scenario

Reject invalid Cookie names before startup side effects.

## Preconditions

In-process run.Run; every scenario is invalid, so it returns before changing auth configuration or starting services.

## Steps

Pass the leaf's invalid option to the selected startup surface.

## Context

No process stdio, environment, working-directory, or live configuration mutation.

```go
import ("testing"; "github.com/xhd2015/doctest/session")
func Setup(t *testing.T, _ *session.Doctest, req *Request) error { req.Mode="reject"; return nil }
```
