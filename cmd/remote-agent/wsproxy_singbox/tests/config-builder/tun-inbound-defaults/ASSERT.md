## Expected

- TUN inbound exists with `auto_route` and `strict_route` both true.

## Side Effects

- None.

## Errors

- None.

## Exit Code

- Success.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.RunErr != nil {
		t.Fatalf("build error: %v", resp.RunErr)
	}
	tun := findTunInbound(resp.ConfigJSON)
	if tun == nil {
		t.Fatal("missing tun inbound")
	}
	if tun["auto_route"] != true {
		t.Fatalf("auto_route = %v, want true", tun["auto_route"])
	}
	if tun["strict_route"] != true {
		t.Fatalf("strict_route = %v, want true", tun["strict_route"])
	}
}
```