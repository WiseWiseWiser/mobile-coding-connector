```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	if req == nil {
		t.Fatal("nil request")
	}
	return nil
}
```
