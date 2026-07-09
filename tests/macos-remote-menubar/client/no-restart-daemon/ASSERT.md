## Expected

1. `RestartDaemonGated` is true — Restart Daemon is absent from remote target
   or only shown when SpawnsDaemon/local profile is active.
2. `HasRestartDaemonMenu` is false — remote must not unconditionally expose
   Restart Daemon / Restart Server.

## Side Effects

- None (read-only source inspection).

## Errors

- Shared menu always shows Restart Daemon for remote product too.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.SourcesChecked) == 0 {
		t.Fatal("no Swift sources found for remote/local menu contract — implement remote product or AppProfile gating")
	}
	if resp.HasRestartDaemonMenu {
		t.Fatalf("remote menu still exposes ungated Restart Daemon (sources: %v)", resp.SourcesChecked)
	}
	if !resp.RestartDaemonGated {
		t.Fatalf("Restart Daemon not gated/absent for remote (sources: %v)", resp.SourcesChecked)
	}
}
```
