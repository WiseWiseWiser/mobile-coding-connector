# Scenario

**Feature**: empty name id

```
empty-name-id -> formatter / client contract
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "format"
	req.SessionID = "D922B298-25FB-41FA-BAF8-7AC7A1D56758"
	return nil
}
```
