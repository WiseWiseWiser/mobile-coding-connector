## Expected

1. Exit 0.
2. Stdout: `token set for <xdev URL> (aliases: xdev)`.
3. Stderr warns that `--token` is visible in history/ps (prefer `--token-stdin`).
4. Stdout never echoes `tok-new`.
5. The client config now stores `tok-new` for the xdev domain, and the default
   domain (plus its `tok-default`) is unchanged.
6. No HTTP request was made (`config set` is local-only).

## Exit Code

0.

```go
import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	if !strings.Contains(resp.Stdout, "token set for "+resp.AliasURL+" (aliases: "+aliasLeafName+")") {
		t.Fatalf("stdout missing the token-set line:\n%s", resp.Stdout)
	}
	if !strings.Contains(resp.Stderr, "warning: --token is visible in shell history and ps; prefer --token-stdin") {
		t.Fatalf("stderr missing the argv-token warning:\n%s", resp.Stderr)
	}
	if strings.Contains(resp.Stdout, "tok-new") {
		t.Fatalf("stdout must never echo the token:\n%s", resp.Stdout)
	}

	var cfg struct {
		Default string `json:"default"`
		Domains []struct {
			Server string `json:"server"`
			Token  string `json:"token"`
		} `json:"domains"`
	}
	if err := json.Unmarshal([]byte(resp.ConfigJSON), &cfg); err != nil {
		t.Fatalf("parse config: %v\n%s", err, resp.ConfigJSON)
	}
	tokens := map[string]string{}
	for _, d := range cfg.Domains {
		tokens[d.Server] = d.Token
	}
	if tokens[resp.AliasURL] != "tok-new" {
		t.Fatalf("alias server token = %q, want tok-new\n%s", tokens[resp.AliasURL], resp.ConfigJSON)
	}
	if tokens[resp.DefaultURL] != aliasDefaultToken {
		t.Fatalf("default token changed to %q, want %s", tokens[resp.DefaultURL], aliasDefaultToken)
	}
	if cfg.Default != resp.DefaultURL {
		t.Fatalf("default domain changed to %q, want %s", cfg.Default, resp.DefaultURL)
	}
	if resp.DefaultHits != 0 || resp.AliasHits != 0 {
		t.Fatalf("config set must be local-only (default=%d alias=%d)", resp.DefaultHits, resp.AliasHits)
	}
}
```
