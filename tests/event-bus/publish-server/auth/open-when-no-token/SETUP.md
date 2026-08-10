# Scenario

**Feature**: open publish when server has no token

```
# no token configured
ServerToken="" + no Authorization -> POST /publish succeeds (2xx)
```

## Steps

1. Clear ServerToken and ClientToken; OmitAuth true.
2. POST fixture event.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.ServerToken = ""
	req.ClientToken = ""
	req.OmitAuth = true
	return nil
}
```
