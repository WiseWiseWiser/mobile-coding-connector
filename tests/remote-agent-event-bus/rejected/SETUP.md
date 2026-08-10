# Scenario

**Feature**: event-bus CLI rejection paths

```
event-bus <bad> -> Error: + non-zero exit
```

## Steps

1. Op=cli; leaf sets bad Args.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.Op = "cli"
	return nil
}
```
