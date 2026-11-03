# Scenario

Reject invalid Cookie configuration at the selected startup boundary.

## Preconditions

No running service or temporary global environment changes.

## Steps

Call the real entrypoint with the invalid option below.

## Context

L2 in-process validation; no service startup.

```go
import ("testing"; "github.com/xhd2015/doctest/session")
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
    req.Args = []string{"keep-alive","--script","--auth-cookie-name",""}
    return nil
}
```
