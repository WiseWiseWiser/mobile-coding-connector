# Scenario

**Feature**: port list help

```
remote-agent port list --help -> list usage + flags
```

## Steps

1. Args: `port list --help`.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "port", "list", "--help")
	return nil
}
```
