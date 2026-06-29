## Expected

- `DomainPolicy.Mode` is whitelist.

```go
import (
	"testing"

	singbox "github.com/xhd2015/ai-critic/cmd/remote-agent/wsproxy_singbox"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.RunErr != nil {
		t.Fatalf("parse error: %v", resp.RunErr)
	}
	if resp.DomainPolicy.Mode != singbox.PolicyWhitelist {
		t.Fatalf("mode = %v, want whitelist", resp.DomainPolicy.Mode)
	}
	if len(resp.DomainPolicy.Include) != 1 || !resp.DomainPolicy.Include[0].Wildcard {
		t.Fatalf("include = %#v", resp.DomainPolicy.Include)
	}
}
```