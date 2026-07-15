## Expected

1. Wrong-process listener remains alive (`ProcessAlive` true).
2. Port still listening (`PortListening` true).
3. PID appears in `KillSkipped` OR `KillKilled` is empty (not killed).

## Errors

- Killing the wrong process fails the test.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ListenerPID <= 0 {
		t.Fatal("wrong-process pid not recorded")
	}
	if !resp.ProcessAlive {
		t.Fatalf("wrong process pid %d was killed", resp.ListenerPID)
	}
	if !resp.PortListening {
		t.Fatal("wrong-process port closed unexpectedly")
	}
	killedWrong := false
	for _, pid := range resp.KillKilled {
		if pid == resp.ListenerPID {
			killedWrong = true
			break
		}
	}
	if killedWrong {
		t.Fatalf("KillOpencodeServePIDs killed non-opencode pid %d", resp.ListenerPID)
	}
}
```
