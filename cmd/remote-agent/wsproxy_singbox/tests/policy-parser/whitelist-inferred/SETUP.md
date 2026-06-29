# Scenario

**Feature**: only --include infers whitelist mode

```go
import (
	"testing"

	singbox "github.com/xhd2015/ai-critic/cmd/remote-agent/wsproxy_singbox"
)

func Setup(t *testing.T, req *Request) error {
	req.PolicyInput = singbox.PolicyInput{Include: []string{"*.corp.com"}}
	return nil
}
```