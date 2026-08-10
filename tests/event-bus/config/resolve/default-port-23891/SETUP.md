# Scenario

**Feature**: default publish port is 23891

```
# zero port flag
DefaultPublishPort() == 23891
ResolvePublishConfig(0,"",false).Port == 23891, Disabled=false
```

## Steps

1. PortFlag=0, empty token, NoPublish=false.
2. Op resolve-config (also captures DefaultPort).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.Op = "resolve-config"
	req.PortFlag = 0
	req.TokenFlag = ""
	req.NoPublish = false
	return nil
}
```
