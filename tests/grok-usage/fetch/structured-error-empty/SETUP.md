# Scenario

**Feature**: fetch failure → no invented structured reset fields

```
FetchMode=fail -> status=error; reset_at/display/time_left empty
```

## Steps

1. Set `FetchMode=fail`.

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
