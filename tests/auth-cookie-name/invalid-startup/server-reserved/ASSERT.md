## Expected

An explicit --auth-cookie-name validation error before startup.

## Side Effects

None.

## Errors

Accepting the invalid name fails this scenario.

## Exit Code

Rejected by the product entrypoint.

```go
import ("testing"; "github.com/xhd2015/doctest/session")
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
    assertRejected(t, resp, err)
}
```
