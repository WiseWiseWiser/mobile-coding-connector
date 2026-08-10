# Scenario

**Feature**: watch short inject stream then exit

```
WatchInject=ok + WatchLines -> agent-run watch sess-w -> lines on stdout, exit 0
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "watch", "sess-w")
	req.Seeds = seedLiveTTY("sess-w", "term-w")
	req.WatchInject = "ok"
	req.WatchLines = []string{"WATCH_A", "WATCH_B"}
	return nil
}
```
