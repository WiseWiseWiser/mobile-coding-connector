## Expected

1. `UsesAuthToken` is `true`.
2. `ConfigFileName` remains `local-agent-config.json`.
3. `SpawnsDaemon` remains `true` (local product still spawns keep-alive).

## Errors

- Leaving `UsesAuthToken=false` after local ServerClient requires Bearer auth.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.UsesAuthToken {
		t.Fatal("Local().UsesAuthToken = false, want true for local Bearer auth")
	}
	if resp.ConfigFileName != "local-agent-config.json" {
		t.Fatalf("ConfigFileName = %q, want local-agent-config.json", resp.ConfigFileName)
	}
	if !resp.SpawnsDaemon {
		t.Fatal("Local().SpawnsDaemon = false, want true")
	}
}
```
