# Scenario

**Feature**: remote no switcher

```
remote-no-switcher -> formatter / client contract
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "client"
	req.ClientLeaf = "remote-no-switcher"
	return nil
}
```
