## Expected

- `DomainPolicy.Mode` is blacklist.

## Exit Code

- Success.

```go
import (
	"testing"

	singbox "github.com/xhd2015/ai-critic/cmd/agentcli/wsproxy_singbox"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.RunErr != nil {
		t.Fatalf("parse error: %v", resp.RunErr)
	}
	if resp.DomainPolicy == nil {
		t.Fatal("missing DomainPolicy")
	}
	if resp.DomainPolicy.Mode != singbox.PolicyBlacklist {
		t.Fatalf("mode = %v, want blacklist", resp.DomainPolicy.Mode)
	}
}
```