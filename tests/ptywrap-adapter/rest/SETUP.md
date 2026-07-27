# Scenario

**Feature**: REST list sessions API shape unchanged after refactor

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Phase = "list-shape"
	return nil
}
```