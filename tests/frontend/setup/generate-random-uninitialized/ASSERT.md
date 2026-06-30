## Expected

1. `Response.ServerStarted` is `true`.
2. `Response.ScriptExitCode` is `0`.
3. `Response.ScriptResult["ok"]` is `true`.
4. `Response.ScriptResult["setupVisible"]` is `true`.
5. `Response.ScriptResult["errorText"]` is empty.
6. `Response.ScriptResult["credentialLength"]` is `64`.

## Side Effects

- Normal-mode server (no quick-test) and Vite are started and torn down.
- Headless Chromium opens the Setup page.

## Errors

- Setup page not shown.
- Error div shows `not_initialized` after clicking Generate Random.
- Credential input remains empty.

```go
import (
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		out := ""
		if resp != nil {
			out = resp.ScriptOutput
		}
		t.Fatalf("Run returned unexpected error: %v\nscript output:\n%s", err, out)
	}

	if !resp.ServerStarted {
		t.Fatal("server did not start successfully")
	}

	if resp.ScriptExitCode != 0 {
		t.Fatalf("playwright script exited with code %d\noutput:\n%s", resp.ScriptExitCode, resp.ScriptOutput)
	}

	if resp.ScriptResult == nil {
		t.Fatalf("no JSON result parsed from script output:\n%s", resp.ScriptOutput)
	}

	setupVisible, _ := resp.ScriptResult["setupVisible"].(bool)
	if !setupVisible {
		t.Fatalf("Setup page not visible: %+v", resp.ScriptResult)
	}

	errText, _ := resp.ScriptResult["errorText"].(string)
	if errText != "" {
		t.Fatalf("BUG: Generate Random showed error %q; want credential in input (root cause: /api/auth/credentials/generate blocked by auth middleware with not_initialized)", errText)
	}

	credLen, _ := resp.ScriptResult["credentialLength"].(float64)
	if int(credLen) != 64 {
		t.Fatalf("expected 64-char credential in input, got length %v result=%+v", credLen, resp.ScriptResult)
	}

	ok, _ := resp.ScriptResult["ok"].(bool)
	if !ok {
		t.Fatalf("ScriptResult.ok is not true: %+v\noutput:\n%s", resp.ScriptResult, resp.ScriptOutput)
	}

	t.Logf("generate random succeeded: credentialLength=%v baseURL=%s", credLen, resp.BaseURL)
}
```