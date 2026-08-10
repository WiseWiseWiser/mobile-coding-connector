# Scenario

**Feature**: --no-event-bus-publish disables publish config

```
# noPublish flag
ResolvePublishConfig(any, any, true) -> Disabled=true
```

## Steps

1. NoPublish=true (port/token irrelevant for Disabled).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.Op = "resolve-config"
	req.PortFlag = 30000
	req.TokenFlag = "ignored-when-disabled"
	req.NoPublish = true
	return nil
}
```
