# Scenario

**Feature**: missing Authorization rejected when token configured

```
# token required
ServerToken=secret + no Authorization header -> 401
```

## Steps

1. ServerToken set; OmitAuth true (no header).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.ServerToken = "test-secret"
	req.ClientToken = ""
	req.OmitAuth = true
	return nil
}
```
