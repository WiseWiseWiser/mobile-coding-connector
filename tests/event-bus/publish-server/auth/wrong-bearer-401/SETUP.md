# Scenario

**Feature**: wrong Bearer rejected when token configured

```
# mismatch
ServerToken=secret + Bearer other -> 401
```

## Steps

1. ServerToken `test-secret`, ClientToken `wrong-token`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.ServerToken = "test-secret"
	req.ClientToken = "wrong-token"
	req.OmitAuth = false
	return nil
}
```
