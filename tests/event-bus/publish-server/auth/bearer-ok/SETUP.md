# Scenario

**Feature**: correct Bearer accepted when token configured

```
# matching token
ServerToken=secret + Authorization Bearer secret -> 2xx
```

## Steps

1. ServerToken and ClientToken both `test-secret`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.ServerToken = "test-secret"
	req.ClientToken = "test-secret"
	req.OmitAuth = false
	return nil
}
```
