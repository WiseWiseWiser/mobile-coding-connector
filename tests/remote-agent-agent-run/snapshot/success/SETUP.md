# Scenario

**Feature**: snapshot success prints inject text

```
SnapshotInject=ok -> agent-run snapshot sess-snap -> inject text
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "snapshot", "sess-snap")
	req.Seeds = seedLiveTTY("sess-snap", "term-snap")
	req.SnapshotInject = "ok"
	req.SnapshotText = "SANITIZED_SNAPSHOT_LINE\n"
	return nil
}
```
