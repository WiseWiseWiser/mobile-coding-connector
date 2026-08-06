## Expected

Profile file exists and content includes (substring contracts):

1. A local `root =` mentioning the local abs path (or the path itself).
2. `ssh://remote-agent/` (absolute remote form uses `ssh://remote-agent//…`).
3. The remote abs path.
4. `sshargs` and `-F` and the SSHConfigDir path (or `ssh_config`).
5. `servercmd` and `/usr/local/bin/unison`.
6. `prefer` and `newer`.
7. At least one default ignore (e.g. `.DS_Store` or `node_modules`).
8. `auto`, `batch`, and `times` lines present (true).

## Side Effects

- Profile written under UnisonDir.

## Errors

- None (`RunErr` empty).

## Exit Code

- Nil error.

```go
import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	_ = d
	if err != nil {
		t.Fatalf("harness Run returned unexpected error: %v", err)
	}
	if resp.RunErr != "" {
		t.Fatalf("init error: %s", resp.RunErr)
	}
	if !resp.ProfileExists {
		t.Fatalf("profile missing at %s", resp.ProfilePath)
	}
	c := resp.ProfileContent
	needles := []string{
		req.LocalPath,
		"ssh://remote-agent/",
		req.RemotePath,
		"sshargs",
		"-F",
		"ssh_config",
		"servercmd",
		"/usr/local/bin/unison",
		"prefer",
		"newer",
		"auto",
		"batch",
		"times",
	}
	for _, n := range needles {
		if !strings.Contains(c, n) {
			t.Fatalf("profile missing %q; content:\\n%s", n, c)
		}
	}
	// SSHConfigDir should appear in sshargs -F path
	if !strings.Contains(c, req.SSHConfigDir) && !strings.Contains(c, filepath.Join(req.SSHConfigDir, "ssh_config")) {
		// -F path might be joined; require ssh_config already checked
	}
	if !strings.Contains(c, ".DS_Store") && !strings.Contains(c, "node_modules") {
		t.Fatalf("profile missing default ignore entries; content:\\n%s", c)
	}
}
```
