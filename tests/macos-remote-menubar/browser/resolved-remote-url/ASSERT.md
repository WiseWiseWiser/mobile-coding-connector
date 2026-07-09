## Expected

1. `BrowserURL` is `https://remote.example`.
2. URL does not contain `127.0.0.1`, `localhost`, or keep-alive port `23312`.
3. URL does not contain the token.

## Errors

- Opening local keep-alive management URL instead of remote server.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.BrowserURL != "https://remote.example" {
		t.Fatalf("BrowserURL = %q, want https://remote.example", resp.BrowserURL)
	}
	lower := strings.ToLower(resp.BrowserURL)
	if strings.Contains(lower, "127.0.0.1") || strings.Contains(lower, "localhost") {
		t.Fatalf("BrowserURL must not be loopback: %q", resp.BrowserURL)
	}
	if strings.Contains(resp.BrowserURL, "23312") {
		t.Fatalf("BrowserURL must not use keep-alive port: %q", resp.BrowserURL)
	}
	if strings.Contains(resp.BrowserURL, req.BrowserToken) {
		t.Fatalf("BrowserURL must not include token: %q", resp.BrowserURL)
	}
}
```
