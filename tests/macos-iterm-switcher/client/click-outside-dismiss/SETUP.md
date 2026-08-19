# Scenario

**Feature**: click outside dismiss

```
click-outside-dismiss -> formatter / client contract
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "client"
	req.ClientLeaf = "click-outside-dismiss"
	return nil
}
```
