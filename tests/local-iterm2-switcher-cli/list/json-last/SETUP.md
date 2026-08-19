# Scenario

**Feature**: list --json prints last inventory JSON without ANSI

```
SeedCache (sess-a)
local-agent native-terminals list --json
  -> stdout = last inventory JSON (sess-a), trailing \n, no ANSI
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Args = []string{"native-terminals", "list", "--json"}
	req.SeedCache = true
	return nil
}
```
