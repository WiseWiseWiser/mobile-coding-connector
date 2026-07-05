## Expected

1. `Env["AI_CRITIC_NO_OPEN_BROWSER"]` is exactly `"1"`.
2. `Env` does **not** contain `CODEX_SHOW_STATUS_BIN`.
3. `Env` does **not** contain `GROK_SHOW_USAGE_BIN`.

## Errors

- Key missing or value not `"1"`.
- Legacy usage-bin env keys still present after in-process refactor.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	got, ok := resp.Env["AI_CRITIC_NO_OPEN_BROWSER"]
	if !ok {
		t.Fatal("AI_CRITIC_NO_OPEN_BROWSER not set in KeepAliveEnv")
	}
	if got != "1" {
		t.Fatalf("AI_CRITIC_NO_OPEN_BROWSER = %q, want %q", got, "1")
	}
	if _, ok := resp.Env["CODEX_SHOW_STATUS_BIN"]; ok {
		t.Fatalf("CODEX_SHOW_STATUS_BIN should not be set; got %q", resp.Env["CODEX_SHOW_STATUS_BIN"])
	}
	if _, ok := resp.Env["GROK_SHOW_USAGE_BIN"]; ok {
		t.Fatalf("GROK_SHOW_USAGE_BIN should not be set; got %q", resp.Env["GROK_SHOW_USAGE_BIN"])
	}
}
```