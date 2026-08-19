# Scenario

**Feature**: inventory capture enriches only when bookmarks need agent ids

```
notes items with complete agent pair -> NeedsAgentEnrich
capture NoEnrich derived from that helper (not hardcoded true)
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
