## Expected

- `slack.mode` is `socket`.
- `slack.dm_policy` is `pairing`.
- `slack.require_mention` is true.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	s := resp.Config.Slack
	if s == nil || !s.Enabled {
		t.Fatal("slack should be enabled")
	}
	if s.Mode != "socket" {
		t.Fatalf("mode = %q, want socket", s.Mode)
	}
	if s.DMPolicy != "pairing" {
		t.Fatalf("dm_policy = %q, want pairing", s.DMPolicy)
	}
	if s.RequireMention == nil || !*s.RequireMention {
		t.Fatalf("require_mention = %v, want true", s.RequireMention)
	}
}
```