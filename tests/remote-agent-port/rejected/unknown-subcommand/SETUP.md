# Scenario

**Feature**: unknown port subcommand

```
port foo -> Error
```

## Steps

1. Args: port foo.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "port", "foo")
	return nil
}
```
