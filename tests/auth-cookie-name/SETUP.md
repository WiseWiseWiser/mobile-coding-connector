# Scenario

Configurable management Cookie namespaces and startup validation.

## Preconditions

Fixture credentials only; no live instances, user credentials, public providers, or existing ports.

## Steps

Run either an invalid startup option in-process, or two isolated auth/proxy fixture processes.

## Context

The per-process auth state makes a process boundary meaningful for the positive namespace case.

```go
import (
    "strings"
    "testing"
)

func assertRejected(t *testing.T, resp *Response, err error) {
    t.Helper()
    if err != nil { t.Fatal(err) }
    if resp == nil || !strings.Contains(resp.Error,"invalid --auth-cookie-name") { t.Fatalf("expected startup validation error, got %+v",resp) }
}
```
