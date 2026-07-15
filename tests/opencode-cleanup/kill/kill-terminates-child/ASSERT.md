## Expected

1. `KillErr` is nil.
2. `FakeOpenCodePID` was killed (`ProcessAlive` false).
3. `PortListening` false on `FakeOpenCodePort`.
4. `FakeOpenCodePID` appears in `KillKilled`.

## Errors

- Child survives or port still open fails the test.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.KillErr != nil {
		t.Fatalf("KillOpencodeServePIDs error: %v", resp.KillErr)
	}
	if resp.FakeOpenCodePID <= 0 {
		t.Fatal("fake opencode pid not recorded")
	}
	if resp.ProcessAlive {
		t.Fatalf("fake opencode pid %d still alive", resp.FakeOpenCodePID)
	}
	if resp.PortListening {
		t.Fatalf("port %d still listening", resp.FakeOpenCodePort)
	}
	killed := false
	for _, pid := range resp.KillKilled {
		if pid == resp.FakeOpenCodePID {
			killed = true
			break
		}
	}
	if !killed {
		t.Fatalf("KillKilled %v missing pid %d", resp.KillKilled, resp.FakeOpenCodePID)
	}
}
```
