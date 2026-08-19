# Scenario

**Feature**: switcher paths are not auth skip-listed

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "skip_list"
	return nil
}
```
