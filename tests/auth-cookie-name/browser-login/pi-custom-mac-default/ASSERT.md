---
label: e2e
---

## Expected

The Pi and Mac sessions stay independent; Mac default-name Cookie passes the Pi origin hop.

## Side Effects

Only temporary binaries, private fixture files, and managed ephemeral fixture processes; all are cleaned up.

## Errors

Leaked Pi credentials, stripped Mac Cookie, or broken legacy/default/Bearer behavior fail.

## Exit Code

Success.

```go
import ("testing"; "github.com/xhd2015/doctest/session")
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
    if err!=nil {t.Fatal(err)}
    expected:=map[string]int{
        "pi-anonymous":401,"mac-anonymous":401,
        "pi-cookie":200,"mac-cookie":200,"mac-refresh":200,
        "pi-legacy-cookie":401,"pi-bearer":200,"mac-bearer":200,
        "pi-bearer-not-forwarded":401,"mac-after-pi-relogin":200,
        "same-host-mac-after-pi-login":200,"same-host-pi-cookie":200,
    }
    for name,want:=range expected {if resp.Checks[name]!=want {t.Errorf("%s: got %d, want %d",name,resp.Checks[name],want)}}
}
```
