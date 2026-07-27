# Scenario

**Feature**: invalid port rejected

```
port visit 0 / 99999 -> Error, non-zero
```

## Steps

1. This leaf checks port 0; high port covered by same validation message class.
   Use port 0.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "cli"
	enableOwnedQuick(req, true, true)
	setCLI(req, "port", "visit", "0")
	return nil
}
```
