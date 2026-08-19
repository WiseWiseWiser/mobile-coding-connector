## Expected

1. Package has a needs-agent helper (`NeedsAgentEnrich` / `needsAgentEnrich`).
2. Capture is not `CaptureOpts{NoEnrich: true}` with no helper.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasNeedsAgentEnrich {
		t.Fatal("want NeedsAgentEnrich / needsAgentEnrich used to decide capture enrich")
	}
	if resp.HardcodedNoEnrichOnly {
		t.Fatal("capture must not hardcode NoEnrich: true when no needs-agent helper exists")
	}
}
```
