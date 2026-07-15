## Expected

1. `CollectErr` is nil.
2. `CollectedPIDs` contains `FakeOpenCodePID`.
3. `FakeOpenCodePort` is listening before Collect (child started in Run).

## Errors

- lsof or Collect failure fails the test.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.CollectErr != nil {
		t.Fatalf("CollectOpencodeServePIDs error: %v", resp.CollectErr)
	}
	if resp.FakeOpenCodePID <= 0 {
		t.Fatal("fake opencode pid not recorded")
	}
	found := false
	for _, pid := range resp.CollectedPIDs {
		if pid == resp.FakeOpenCodePID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("CollectedPIDs %v missing fake opencode pid %d", resp.CollectedPIDs, resp.FakeOpenCodePID)
	}
}
```
