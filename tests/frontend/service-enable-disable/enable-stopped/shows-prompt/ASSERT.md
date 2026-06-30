---
label: ui-automation
explanation: Playwright drives /home/service Enable flow; quick-test compile ~25s
---

## Expected

1. `Response.ServerStarted` is true.
2. `Response.ScriptExitCode` is 0 and `ScriptResult.ok` is true.
3. `ScriptResult.modalText` contains `daemon` and `next` (case-insensitive).
4. `ScriptResult.enabledUi` is true — Disable button visible or disabled badge cleared.
5. `ScriptResult.apiEnabled` is true after confirm.

## Side Effects

- Enable persists `enabled=true` and schedules daemon start.

## Errors

- Enable button or modal missing.
- Modal lacks daemon-check message.
- UI does not reflect enabled state.

## Exit Code

0.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/assert"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		out := ""
		if resp != nil {
			out = resp.ScriptOutput
		}
		t.Fatalf("Run error: %v\nscript output:\n%s", err, out)
	}
	if resp.SeedError != "" {
		t.Fatalf("service seed failed: %s", resp.SeedError)
	}
	if !resp.ServerStarted {
		t.Fatal("quick-test server did not start")
	}
	if resp.ScriptExitCode != 0 {
		t.Fatalf("script exit %d\n%s", resp.ScriptExitCode, resp.ScriptOutput)
	}
	if resp.ScriptResult == nil {
		t.Fatalf("no ScriptResult parsed\n%s", resp.ScriptOutput)
	}

	ok, _ := resp.ScriptResult["ok"].(bool)
	if !ok {
		t.Fatalf("ScriptResult.ok false: %+v\n%s", resp.ScriptResult, resp.ScriptOutput)
	}

	modalText, _ := resp.ScriptResult["modalText"].(string)
	msg := strings.ToLower(modalText)
	assert.Output(t, msg, `<contains>
daemon
next
</contains>`)

	enabledUI, _ := resp.ScriptResult["enabledUi"].(bool)
	if !enabledUI {
		t.Fatalf("UI did not reflect enabled state: %+v", resp.ScriptResult)
	}

	apiEnabled, ok := resp.ScriptResult["apiEnabled"].(bool)
	if !ok || !apiEnabled {
		t.Fatalf("apiEnabled = %v ok=%v, want true", resp.ScriptResult["apiEnabled"], ok)
	}
}
```