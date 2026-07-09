## Expected

1. `SpawnsDaemon` is `true`.
2. `ConfigFileName` is not `remote-agent-config.json` (local uses local config or none for CLI isolation — prefer `local-agent-config.json` or empty app-local path as implementer defines; must not be the remote CLI file).
3. `AppName` is `ai-critic-macos` when set.
4. `BundleID` is `com.xhd2015.ai-critic-macos` when set.

## Errors

- Local profile flipped to no-daemon by remote work.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.SpawnsDaemon {
		t.Fatal("local profile must SpawnsDaemon=true")
	}
	if resp.ConfigFileName == "remote-agent-config.json" {
		t.Fatal("local profile must not use remote-agent-config.json")
	}
	// Prefer explicit local identity when fields are populated.
	if resp.AppName != "" && resp.AppName != "ai-critic-macos" {
		t.Fatalf("AppName = %q, want ai-critic-macos (or empty if not applicable)", resp.AppName)
	}
	if resp.BundleID != "" && resp.BundleID != "com.xhd2015.ai-critic-macos" {
		t.Fatalf("BundleID = %q, want com.xhd2015.ai-critic-macos (or empty)", resp.BundleID)
	}
}
```
