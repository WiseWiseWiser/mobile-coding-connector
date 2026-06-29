# Scenario

**Feature**: whitelist config omits catch-all and routes include patterns to web

```go
import (
	"testing"

	singbox "github.com/xhd2015/ai-critic/cmd/remote-agent/wsproxy_singbox"
)

func Setup(t *testing.T, req *Request) error {
	req.Op = OpBuildHttpOnlyConfig
	req.HttpOnlySocksPort = 11080
	policy, err := singbox.ParseDomainPolicy(singbox.PolicyInput{Include: []string{"*.corp.com"}})
	if err != nil {
		return err
	}
	req.HttpOnlyPolicy = policy
	return nil
}
```