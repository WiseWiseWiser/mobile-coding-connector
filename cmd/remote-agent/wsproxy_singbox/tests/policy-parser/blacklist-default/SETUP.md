# Scenario

**Feature**: empty include/exclude defaults to blacklist mode

```go
import (
	"testing"

	singbox "github.com/xhd2015/ai-critic/cmd/remote-agent/wsproxy_singbox"
)

func Setup(t *testing.T, req *Request) error {
	req.PolicyInput = singbox.PolicyInput{}
	return nil
}
```