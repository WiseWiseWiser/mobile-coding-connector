# Scenario

**Feature**: port 99999 rejected as invalid

```
port visit 99999 -> Error
```

## Steps

1. Args visit 99999.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "cli"
	enableOwnedQuick(req, true, true)
	setCLI(req, "port", "visit", "99999")
	return nil
}
```
