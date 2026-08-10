## Expected

1. WebSocket attach succeeds (upgrade 101 or ExitCode 0 with empty AttachErr).
2. `ReceivedOutput` contains the echoed payload `hello-attach` (binary/text
   frame roundtrip through FakeAttach).
3. No fatal hang — Run returns.

## Errors

- Upgrade failure.
- Empty output without echo.

## Exit Code

0 on success path

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.AttachErr != "" && resp.ExitCode != 0 {
		t.Fatalf("live attach failed: status=%d err=%q body=%q", resp.WSHTTPStatus, resp.AttachErr, resp.Body)
	}
	if resp.WSHTTPStatus != 0 && resp.WSHTTPStatus != 101 {
		t.Fatalf("want WS upgrade 101, got %d err=%q", resp.WSHTTPStatus, resp.AttachErr)
	}
	if !strings.Contains(resp.ReceivedOutput, "hello-attach") {
		t.Fatalf("expected echoed payload in ReceivedOutput; got %q err=%q", resp.ReceivedOutput, resp.AttachErr)
	}
}
```
