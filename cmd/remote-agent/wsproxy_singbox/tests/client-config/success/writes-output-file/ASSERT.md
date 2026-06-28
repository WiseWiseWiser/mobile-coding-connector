## Expected

- `Run` succeeds.
- Output file exists and contains valid JSON with tun inbound.
- Stdout is empty or whitespace-only.

## Side Effects

- `singbox-client-config.json` created alongside the leaf.

## Errors

- None.

## Exit Code

- Success.

```go
import (
	"encoding/json"
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.RunErr != nil {
		t.Fatalf("client-config error: %v", resp.RunErr)
	}
	if len(resp.OutputData) == 0 {
		t.Fatal("output file is empty")
	}
	var cfg map[string]any
	if err := json.Unmarshal(resp.OutputData, &cfg); err != nil {
		t.Fatalf("output file is not valid JSON: %v", err)
	}
	if !configHasTunInbound(cfg) {
		t.Fatalf("missing tun inbound in output file")
	}
	if strings.TrimSpace(resp.Stdout) != "" {
		t.Fatalf("stdout should be empty with --output; got %q", resp.Stdout)
	}
}
```