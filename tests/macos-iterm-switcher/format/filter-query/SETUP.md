# Scenario

**Feature**: filter query

```
filter-query -> formatter / client contract
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "format"
	req.SidebarID = "all"
	req.Query = "auth"
	return nil
}
```
