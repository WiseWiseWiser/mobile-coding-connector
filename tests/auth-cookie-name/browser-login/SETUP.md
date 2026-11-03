# Scenario

Independent Pi and Mac browser authentication.

## Preconditions

Separate processes and credential files; independent pi.test and mac.test cookie domains.

## Steps

Log in to both instances and send the Mac session through the production origin hop.

## Context

No full-server background services or live deployment.

```go
import ("testing"; "github.com/xhd2015/doctest/session")
func Setup(t *testing.T, _ *session.Doctest, req *Request) error { req.Mode="namespaces"; return nil }
```
