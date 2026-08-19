```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "client"
	req.ClientLeaf = "no-standalone-settings"
	return nil
}
```
