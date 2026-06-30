---
label: ui-automation
explanation: Playwright drives /home/service Disable flow; quick-test compile ~25s
---

## Expected

1. `Response.ServerStarted` is true.
2. `Response.ScriptExitCode` is 0 and `ScriptResult.ok` is true.
3. `ScriptResult.modalText` contains `won't stop immediately` (case-insensitive).
4. `ScriptResult.stillRunning` is true — API reports running/starting with `apiPid > 0`.
5. After disable, `ScriptResult.apiEnabled` is false when exposed by API.

## Side Effects

- Disable persists `enabled=false` without stopping the process.

## Errors

- Disable button or modal missing.
- Service stops after disable confirm.
- Wrong modal message.

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
	assert.Output(t, strings.ToLower(modalText), `<contains>
won't stop immediately
</contains>`)

	stillRunning, _ := resp.ScriptResult["stillRunning"].(bool)
	if !stillRunning {
		t.Fatalf("service not running after disable confirm: %+v", resp.ScriptResult)
	}

	if enabled, ok := resp.ScriptResult["apiEnabled"].(bool); ok && enabled {
		t.Fatalf("apiEnabled still true after disable: %+v", resp.ScriptResult)
	}
}
```