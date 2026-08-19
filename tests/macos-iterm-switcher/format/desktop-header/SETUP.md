# Scenario

**Feature**: desktop header

```
desktop-header -> formatter / client contract
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "format"
	req.SpaceIndex = 2
	return nil
}
```
