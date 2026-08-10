# Scenario

**Feature**: kill missing session id / unknown session

```
agent-run kill  -> error (no id)
```

Also covers missing positional via validation; inject not required.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "kill")
	return nil
}
```
