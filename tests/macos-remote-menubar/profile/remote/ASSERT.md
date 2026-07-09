## Expected

1. `SpawnsDaemon` is `false`.
2. `UsesAuthToken` is `true`.
3. `ConfigFileName` is `remote-agent-config.json`.
4. `AppName` is `ai-critic-remote-macos`.
5. `BundleID` is `com.xhd2015.ai-critic-remote-macos`.
6. `DisplayName` is `AI Critic(Remote)` (or at least contains `Remote`).

## Errors

- Remote profile still spawning keep-alive daemon or using local config file.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.SpawnsDaemon {
		t.Fatal("remote profile must not SpawnsDaemon")
	}
	if !resp.UsesAuthToken {
		t.Fatal("remote profile must UsesAuthToken")
	}
	if resp.ConfigFileName != "remote-agent-config.json" {
		t.Fatalf("ConfigFileName = %q, want remote-agent-config.json", resp.ConfigFileName)
	}
	if resp.AppName != "ai-critic-remote-macos" {
		t.Fatalf("AppName = %q, want ai-critic-remote-macos", resp.AppName)
	}
	if resp.BundleID != "com.xhd2015.ai-critic-remote-macos" {
		t.Fatalf("BundleID = %q, want com.xhd2015.ai-critic-remote-macos", resp.BundleID)
	}
	if resp.DisplayName != "AI Critic(Remote)" && !strings.Contains(resp.DisplayName, "Remote") {
		t.Fatalf("DisplayName = %q, want AI Critic(Remote) or containing Remote", resp.DisplayName)
	}
}
```
