# Scenario

**Feature**: empty name cwd

```
empty-name-cwd -> formatter / client contract
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "format"
	req.Name = "  "
	req.Cwd = "~/proj/ai-critic"
	req.SessionID = "aaaa-bbbb"
	return nil
}
```
