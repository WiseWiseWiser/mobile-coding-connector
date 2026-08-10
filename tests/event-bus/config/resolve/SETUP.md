# Scenario

**Feature**: ResolvePublishConfig + DefaultPublishPort

```
# flag → PublishConfig
ResolvePublishConfig(port, token, noPublish) -> Port/Token/Disabled
```

## Preconditions

`Op=resolve-config` (or default-port for constant-only leaf).

## Steps

1. Default Op resolve-config.
2. Leaf sets flag fields.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	if req.Op == "" {
		req.Op = "resolve-config"
	}
	return nil
}
```
