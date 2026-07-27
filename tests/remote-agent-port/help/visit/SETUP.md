# Scenario

**Feature**: port visit help

```
remote-agent port visit --help -> provider/idle/detach flags
```

## Steps

1. Args: `port visit --help`.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "port", "visit", "--help")
	return nil
}
```
