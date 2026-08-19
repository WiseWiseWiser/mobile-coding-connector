# Scenario

**Feature**: local/remote Swift client contracts

```
local Shared sources -> inventory/focus/notes paths
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "client"
	return nil
}
```
