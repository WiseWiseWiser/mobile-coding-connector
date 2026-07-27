# Scenario

**Feature**: port list table with listeners

```
seeded local ports -> port list -> PORT/PID/COMMAND rows
```

## Steps

1. Seed one listener; Args `port list`.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "port", "list")
	seedOneListener(req)
	return nil
}
```
