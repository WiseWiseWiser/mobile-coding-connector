# Scenario

**Feature**: default capture NoEnrich comes from a needs-agent helper

```
read server/localiterm2 -> needsAgentEnrich
CaptureOpts.NoEnrich uses that helper, not a hardcoded true
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "enrich_source"
	return nil
}
```
