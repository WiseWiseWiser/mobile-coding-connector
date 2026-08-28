# Scenario

**Feature**: injected error fetcher → service error status

```
FetchMode=fail -> status error
```

## Steps

1. `FetchMode=fail`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.FetchMode = "fail"
	return nil
}
```
