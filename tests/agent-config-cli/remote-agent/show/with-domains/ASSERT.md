## Expected

1. Exit 0.
2. Stdout pretty JSON matches seeded default, domains, and full tokens (no redaction).

## Side Effects

Config file on disk unchanged by --show.

## Errors

Missing domain, redacted token, non-zero exit.

## Exit Code

0.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	assertExitZero(t, resp)
	assertNoConfigUI(t, resp)
	assertPrettyConfigMatches(t, resp.Stdout, sampleRemoteConfig())
}
```
