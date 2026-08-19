# Scenario

**Feature**: local has switcher

```
local-has-switcher -> formatter / client contract
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "client"
	req.ClientLeaf = "local-has-switcher"
	return nil
}
```
