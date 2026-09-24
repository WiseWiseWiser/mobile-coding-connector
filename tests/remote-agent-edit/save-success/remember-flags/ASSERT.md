## Expected Output

Run 1 (`--editor <script> --remember-flags`):

```
Remembered flags for edit: --work-dir __STAGING__ --editor __EDITOR__
Downloading notes.md -> __STAGED__ (4 B)
Opening __EDITOR__ __STAGED__
Saved __REMOTE__ (6 B, md5 __MD5__)
```

Run 2 (`edit notes.md`, no editor flag):

```
Using remembered flags: --work-dir __STAGING__ --editor __EDITOR__
Downloading notes.md -> __STAGED__ (6 B)
Opening __EDITOR__ __STAGED__
Saved __REMOTE__ (7 B, md5 __MD5B__)
```

## Expected

1. Both runs exit 0.
2. Run 1 prints `Remembered flags for edit:` including `--editor <script>`, then `Saved`.
3. Run 2 prints `Using remembered flags:` including the same `--editor`, and saves again.
4. Remote `notes.md` contains `second\n` after run 2 — only possible if the
   remembered script ran (a `vim` fallback fails the terminal guard).
5. The CLI config file records the editor flag.

## Side Effects

- `~/.ai-critic/remote-agent-config.json` gains
  `remembered_flags.edit = ["--work-dir", …, "--editor", …]`.

## Errors

- Run 2 falls back to `vim` (`Error: editor "vim" needs a terminal on stdin`).
- `Using remembered flags` missing, or the second save reports `file not changed`.

## Exit Code

0 (both runs).

```go
import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"

	"github.com/xhd2015/doctest/assert"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	assertExit(t, resp, 0)
	assertSecondExit(t, resp, 0)

	editorFlag := "--editor " + resp.EditorPath
	combinedHasAll(t, resp.Combined,
		"Remembered flags for edit:",
		editorFlag,
		"Saved "+resp.RemotePath,
	)
	combinedHasAll(t, resp.SecondCombined,
		"Using remembered flags:",
		editorFlag,
		"Saved "+resp.RemotePath,
	)
	combinedHasNone(t, resp.SecondCombined, "needs a terminal", "file not changed", "Error")

	assertRemoteContent(t, resp, "notes.md", "second\n")

	configPath := filepath.Join(resp.AgentHome, ".ai-critic", "remote-agent-config.json")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read CLI config %s: %v", configPath, err)
	}
	if !strings.Contains(string(configData), "--editor") || !strings.Contains(string(configData), resp.EditorPath) {
		t.Fatalf("config does not remember the editor flag:\n%s", configData)
	}

	assert.Output(t, resp.Stdout, `---
version: 2
__STAGING__: type=string
__EDITOR__: type=string
__STAGED__: type=string
__REMOTE__: type=string
__MD5__: type=string
---
Remembered flags for edit: --work-dir __STAGING__ --editor __EDITOR__
Downloading notes.md -> __STAGED__ (4 B)
Opening __EDITOR__ __STAGED__
Saved __REMOTE__ (6 B, md5 __MD5__)
`)

	assert.Output(t, resp.SecondStdout, `---
version: 2
__STAGING__: type=string
__EDITOR__: type=string
__STAGED__: type=string
__REMOTE__: type=string
__MD5B__: type=string
---
Using remembered flags: --work-dir __STAGING__ --editor __EDITOR__
Downloading notes.md -> __STAGED__ (6 B)
Opening __EDITOR__ __STAGED__
Saved __REMOTE__ (7 B, md5 __MD5B__)
`)
}
```
