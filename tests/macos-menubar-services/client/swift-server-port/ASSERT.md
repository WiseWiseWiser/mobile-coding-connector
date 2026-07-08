## Expected

1. `GrokUsesServerPort` is `true` — grok usage fetched via `ServerClient` on port `23712`.
2. `CodexUsesServerPort` is `true` — codex usage fetched via `ServerClient` on port `23712`.
3. `ServicesUsesAllQuery` is `true` — services list uses `GET /api/services?all=1`.
4. `DaemonPortForGrok` is `false` — `DaemonClient` must not serve grok usage.
5. `ServerPort` equals `23712`.

## Side Effects

- None (read-only source inspection).

## Errors

- Grok/codex/services still target daemon port `23312`.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/server/config"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ServerPort != config.DefaultServerPort {
		t.Fatalf("ServerPort = %d, want %d", resp.ServerPort, config.DefaultServerPort)
	}
	if !resp.GrokUsesServerPort {
		t.Fatalf("grok usage not on ServerClient:%d (sources: %v)", config.DefaultServerPort, resp.SwiftSourcesChecked)
	}
	if !resp.CodexUsesServerPort {
		t.Fatalf("codex usage not on ServerClient:%d (sources: %v)", config.DefaultServerPort, resp.SwiftSourcesChecked)
	}
	if !resp.ServicesUsesAllQuery {
		t.Fatalf("services list missing ?all=1 (sources: %v)", resp.SwiftSourcesChecked)
	}
	if resp.DaemonPortForGrok {
		t.Fatal("DaemonClient still serves /api/grok/usage — business APIs must use server port")
	}
}
```