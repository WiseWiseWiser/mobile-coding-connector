# Scenario

**Feature**: bad flag on port list

```
port list --not-a-flag -> Error
```

## Steps

1. Args: port list --not-a-flag.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "port", "list", "--not-a-flag")
	return nil
}
```
