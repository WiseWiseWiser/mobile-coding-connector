## Expected

1. Exit code is zero.
2. `local-agent-config.json` exists.
3. The `http://localhost:24888` domain row stores `local-import-token`.
4. The imported server is the selected default.
5. Existing unrelated domain rows and `project_bindings` are preserved.
6. Output does not print `local-import-token` or the ignored second token.

## Side Effects

Writes only the local-agent config file under isolated `HOME/.ai-critic`.

## Errors

- Missing config write.
- Wrong token line selected.
- Existing unrelated config data lost.
- Raw credential leaked to stdout/stderr.

## Exit Code

0.

```go
import (
	"encoding/json"
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("expected import-local success, exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	if strings.Contains(resp.Combined, "local-import-token") || strings.Contains(resp.Combined, "second-token-ignored") {
		t.Fatalf("raw credential must not be printed; combined:\n%s", resp.Combined)
	}
	if len(resp.LocalConfigAfter) == 0 {
		t.Fatalf("expected local-agent-config.json to be written at %s", resp.LocalConfigPath)
	}
	var cfg LocalAgentConfigFile
	if err := json.Unmarshal(resp.LocalConfigAfter, &cfg); err != nil {
		t.Fatalf("parse local config: %v\n%s", err, resp.LocalConfigAfter)
	}
	const importedServer = "http://localhost:24888"
	if cfg.Default != importedServer {
		t.Fatalf("default = %q, want %q; config:\n%s", cfg.Default, importedServer, resp.LocalConfigAfter)
	}
	var imported, old bool
	for _, d := range cfg.Domains {
		switch d.Server {
		case importedServer:
			imported = true
			if d.Token != "local-import-token" {
				t.Fatalf("imported token = %q, want first non-empty token", d.Token)
			}
		case "https://old.example.com":
			old = true
			if d.Token != "old-token" {
				t.Fatalf("old domain token changed to %q", d.Token)
			}
		}
	}
	if !imported {
		t.Fatalf("missing imported domain %q; config:\n%s", importedServer, resp.LocalConfigAfter)
	}
	if !old {
		t.Fatalf("existing unrelated domain not preserved; config:\n%s", resp.LocalConfigAfter)
	}
	if len(cfg.ProjectBindings) != 1 ||
		cfg.ProjectBindings[0].Server != "https://old.example.com" ||
		cfg.ProjectBindings[0].RemoteDir != "/remote/project" ||
		cfg.ProjectBindings[0].LocalPath != "/local/project" {
		t.Fatalf("project_bindings not preserved: %+v", cfg.ProjectBindings)
	}
}
```
