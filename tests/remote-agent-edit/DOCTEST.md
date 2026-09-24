# Remote-Agent Edit Doctests

Doctests for `remote-agent edit <REMOTE_PATH> [--editor=CMD] [--remember-flags]
[--work-dir DIR]`: download the remote file into `<work-dir>/<remote-path>`, open
it with a local editor, and write it back through `POST /api/files/write` using
the download-time md5 as precondition.

Most leaves are **L2 in-process** (`fileupload.RegisterAPIForHome` +
`agentcli.RunWithWriters`) with a generated fake editor. Two sparse **L3 e2e**
smokes (2/21 leaves, inside the 10% budget) keep the product binary path:
round trip + terminal-editor tty guard.

# DSN (Domain Specific Notion)

Most leaves are **L2 in-process**: `fileupload.RegisterAPIForHome` on a local mux
+ `agentcli.RunWithWriters` (no product binaries). Two **L3 e2e** smokes keep the
binary path (`UseCLI` + `label: heavy, e2e`): `e2e/roundtrip` and
`save-rejected/terminal-editor-no-tty` (needs a real non-tty stdin).

**L2** serves check/download/write/home APIs in-process via
`RegisterAPIForHome(mux, serverHome)`, captures output through
`agentcli.RunWithWriters`, and scopes CLI config with
`testhooks.SetHomeOverride` (no process stdio/env/cwd mutation). **L3** runs
`ai-critic-server` with `HOME=serverHome` plus `remote-agent` with `HOME=agentHome`
(`cmd.Stdin = nil` → /dev/null, so terminal-editor guards are deterministic).

**Participants**

- **L2: agentcli.RunWithWriters** — in-process `edit` CLI against the local mux.
- **L2: fileupload HTTP** — `RegisterAPIForHome` on ephemeral port.
- **L3: ai-critic-server + remote-agent subprocesses** — the two `UseCLI` smokes.
- **serverHome** — temp fake server home; leaf setup pre-creates remote fixtures.
- **agentHome** — temp home for CLI config (`--remember-flags`) and L3 credentials.
- **agentWorkDir / stagingDir** — per-leaf staging root passed via `--work-dir`;
  the staged copy lives at `<stagingDir>/<absolute remote path>`.
- **session cache** — doctest-injected `DOCTEST_SESSION_ID` keys
  `$TMPDIR/remote-agent-edit-doctest-<id>/` for L3 shared binaries (file lock).

**Behaviors**

- A missing remote file is staged as an empty file; saving creates it (parent
  directories included) and reports `Created` instead of `Saved`.
- The staged copy is `0600` under a `0700` staging root; the remote file keeps its
  mode, and a symlinked remote path is written through (link preserved, warning on
  stderr).
- The base md5 is taken after a **fresh** download (`DownloadOptions.NoResume`), so
  a same-size staged copy from an earlier run can never become the base.
- Unchanged staged content short-circuits with `file not changed` (exit 0, no
  request); an editor exit ≠ 0 aborts without uploading.
- A rejected write (409) keeps the staged copy and prints the conflict recipe
  (`download` → merge → `upload`); when the remote already equals the staged
  content there is nothing to resolve and the run exits 0.
- `--remember-flags` persists `--editor`/`--work-dir` in
  `~/.ai-critic/remote-agent-config.json`; later runs replay them (explicit flags win).
- Terminal editors (`vim`, `nano`, …) require a tty on stdin and fail fast
  otherwise; missing editor binaries fail with an install hint.

## Version

0.0.1

## Decision Tree

```
[remote-agent edit REMOTE_PATH]
 |
 +-- save-success/                       (GROUP) staged edit is written back
 |    +-- edited-file/                   (LEAF)  remote replaced, Saved
 |    +-- new-file/                      (LEAF)  absent remote -> Created (parents made)
 |    +-- no-change/                     (LEAF)  no-op editor -> "file not changed"
 |    +-- reverted-content/              (LEAF)  identical rewrite -> "file not changed"
 |    +-- fresh-base/                    (LEAF)  stale same-size staged copy refreshed
 |    +-- editor-with-args/              (LEAF)  --editor="script --mark" arg split
 |    +-- converged-md5/                 (LEAF)  409 but remote == staged -> exit 0
 |    +-- symlink-target/                (LEAF)  write through symlink + warning
 |    +-- preserves-mode/                (LEAF)  0755 remote stays 0755
 |    +-- remote-empty-file/             (LEAF)  existing empty file -> Saved, not Created
 |    +-- dotfile/                       (LEAF)  .env staged and saved
 |    +-- tilde-path/                    (LEAF)  ~/notes.md resolved before staging
 |    +-- remember-flags/                (LEAF)  --remember-flags replayed next run
 |
 +-- save-rejected/                      (GROUP) the write does not happen
 |    +-- conflict-md5/                  (LEAF)  remote changed -> 409 recipe
 |    +-- conflict-deleted/              (LEAF)  remote deleted -> 409 (+ deletion line)
 |    +-- editor-nonzero/                (LEAF)  editor exit 3 -> nothing uploaded
 |    +-- editor-missing/                (LEAF)  unknown editor -> install hint
 |    +-- remote-is-dir/                 (LEAF)  directory target refused
 |    +-- staged-file-deleted/           (LEAF)  editor removed staged copy -> Error
 |    +-- terminal-editor-no-tty/        (LEAF)  vim without tty -> hint, not launched
 |
 +-- e2e/                                (GROUP) product binaries
      +-- roundtrip/                     (LEAF)  L3 smoke: server + CLI edit
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `save-success/edited-file` | Remote file edited and saved; staged copy kept |
| 2 | `save-success/new-file` | Missing remote file staged empty, then created |
| 3 | `save-success/no-change` | Unchanged content → `file not changed`, no write |
| 4 | `save-success/fresh-base` | Same-size stale staged copy never becomes the base |
| 5 | `save-success/editor-with-args` | `--editor` value with arguments |
| 6 | `save-success/converged-md5` | Remote already matches → exit 0, nothing written |
| 7 | `save-success/symlink-target` | Symlink preserved, target rewritten, warning |
| 8 | `save-success/symlink-target` | Symlink preserved, target rewritten, warning |
| 9 | `save-success/preserves-mode` | 0755 remote file stays executable |
| 10 | `save-success/remote-empty-file` | Existing empty file → `Saved`, not `Created` |
| 11 | `save-success/dotfile` | Dotfile staged and saved |
| 12 | `save-success/tilde-path` | `~/path` resolved against the server home |
| 13 | `save-success/remember-flags` | Remembered editor replayed on the next run |
| 14 | `save-rejected/conflict-md5` | md5 conflict → 409, recipe, staged kept |
| 15 | `save-rejected/conflict-deleted` | Remote deleted during edit → 409, no resurrection |
| 16 | `save-rejected/editor-nonzero` | Editor abort → nothing uploaded |
| 17 | `save-rejected/editor-missing` | Missing editor binary → hint, no write |
| 18 | `save-rejected/remote-is-dir` | Directory target refused before download |
| 19 | `save-rejected/staged-file-deleted` | Staged copy removed by the editor → Error |
| 20 | `save-rejected/terminal-editor-no-tty` | `vim` without tty → hint, never launched (L3) |
| 21 | `e2e/roundtrip` | L3: product server + `remote-agent edit` |

## Parameter Coverage

| Factor (significance →) | Leaves |
|-------------------------|--------|
| Write outcome (200 / 409 / no request) | edited-file, conflict-md5, no-change |
| Remote state (exists / empty / absent / changed / deleted / directory) | edited-file, remote-empty-file, new-file, conflict-md5, conflict-deleted, remote-is-dir |
| Base freshness (fresh / stale same-size / converged / reverted) | edited-file, fresh-base, converged-md5, reverted-content |
| Editor behavior (write / no-op / abort / missing / needs tty / args / removes staged) | edited-file, no-change, editor-nonzero, editor-missing, terminal-editor-no-tty, editor-with-args, staged-file-deleted |
| Metadata (mode / symlink) | preserves-mode, symlink-target |
| Path shape (regular / dotfile / nested new / `~/` / symlink) | edited-file, dotfile, new-file, tilde-path, symlink-target |
| Persistence (`--remember-flags`) | remember-flags |
| Transport (L2 in-process vs L3 binaries) | all leaves, `e2e/roundtrip`, `terminal-editor-no-tty` |

## How to Run

```sh
go run ./script/build                                    # frontend dist + server (L3 needs it)
doctest vet ./tests/remote-agent-edit
doctest test ./tests/remote-agent-edit/...               # L2 mass
doctest test --label e2e ./tests/remote-agent-edit/...   # 2 L3 smokes
```

```go
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/cmd/agentcli"
	"github.com/xhd2015/ai-critic/cmd/agentcli/testhooks"
	"github.com/xhd2015/ai-critic/script/lib"
	"github.com/xhd2015/ai-critic/server/fileupload"
	"github.com/xhd2015/doctest/session"
)

// agentcliInProcessMu serializes in-process agentcli runs: the home override
// installed by testhooks is process-wide state.
var agentcliInProcessMu sync.Mutex

type Request struct {
	Args   []string
	Server string
	Token  string

	// UseCLI forces the L3 product-binary path. Default false → L2 in-process.
	UseCLI bool
	E2E    bool

	// FileArg is the CLI <REMOTE_PATH> argument.
	FileArg string
	// RemoteRel is the serverHome-relative path of the edited file. Defaults
	// to FileArg when FileArg is relative.
	RemoteRel string

	// EditorFlag, when set, is passed verbatim as --editor (raw values such as
	// a missing binary or "vim").
	EditorFlag string
	// EditorWrite is written to the staged file by the generated editor script.
	EditorWrite string
	// EditorWriteSecond is used from the second editor invocation on (lets one
	// script change content again on a later run).
	EditorWriteSecond string
	// EditorDeleteStaged makes the editor remove the staged copy (simulates an
	// editor that writes a temp file and loses the original).
	EditorDeleteStaged bool
	// EditorNoop runs an editor that exits 0 without touching the staged file.
	EditorNoop bool
	// EditorExitCode makes the generated editor exit with this status.
	EditorExitCode int
	// EditorArgsSuffix is appended to the generated editor command so the
	// --editor value carries arguments (e.g. "--mark").
	EditorArgsSuffix string
	// RememberFlags adds --remember-flags to the args.
	RememberFlags bool
	// SecondArgs, when set, runs the CLI again with the same server/HOME. The
	// second run never receives an injected --editor, so it exercises whatever
	// the CLI falls back to (e.g. remembered flags).
	SecondArgs []string

	// RemoteWrite maps serverHome-relative paths to content the editor writes
	// directly (simulates a concurrent remote change).
	RemoteWrite map[string]string
	// RemoteDelete lists serverHome-relative paths the editor removes.
	RemoteDelete []string

	// ServerPreseedFiles maps serverHome-relative paths to file contents.
	ServerPreseedFiles map[string]string
	// ServerPreseedDirs lists empty directories to create under serverHome.
	ServerPreseedDirs []string
	// ServerPreseedModes maps serverHome-relative paths to file modes (0644 default).
	ServerPreseedModes map[string]os.FileMode
	// ServerSymlinks maps a serverHome-relative link to a serverHome-relative target.
	ServerSymlinks map[string]string

	// StaleStaged seeds the staged path with same-size, different content
	// before the run (regression guard for the resume/skip download path).
	StaleStaged bool

	// FakeTerminalEditor installs a fake executable with this name (e.g. "vim")
	// under the work dir and returns its absolute path as --editor. The fake
	// records whether it ran, so a leaf can assert the tty guard refused first.
	// Such leaves must use the L3 binary path, where stdin is /dev/null.
	FakeTerminalEditor string
}

func useBinaryPath(req *Request) bool {
	return req != nil && (req.UseCLI || req.E2E)
}

type Response struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Combined string

	ServerURL  string
	ServerHome string
	AgentHome  string
	AgentWork  string
	StagingDir string

	RemoteRel  string
	RemotePath string
	StagedPath string
	EditorPath string

	// EditorRan reports that the fake terminal editor executable was invoked.
	EditorRan bool

	SecondExitCode int
	SecondStdout   string
	SecondStderr   string
	SecondCombined string
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	resp := &Response{}

	if len(req.Args) == 0 {
		return nil, fmt.Errorf("Request.Args is required (e.g. edit <remote> [options])")
	}
	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	if req.RemoteRel == "" {
		req.RemoteRel = strings.TrimPrefix(filepath.ToSlash(req.FileArg), "/")
	}

	moduleRoot := filepath.Clean(filepath.Join(d.DOCTEST_ROOT, "..", ".."))
	cacheDir := sessionCacheDir(d.DOCTEST_SESSION_ID)

	serverHome, err := os.MkdirTemp("", "edit-server-home-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(serverHome) })
	resp.ServerHome = serverHome

	if err := applyServerPreseed(t, serverHome, req); err != nil {
		return nil, err
	}

	agentHome, err := os.MkdirTemp("", "edit-agent-home-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(agentHome) })
	resp.AgentHome = agentHome

	agentWorkDir, err := os.MkdirTemp("", "edit-agent-work-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(agentWorkDir) })
	resp.AgentWork = agentWorkDir

	stagingDir := filepath.Join(agentWorkDir, "staging")
	if err := os.MkdirAll(stagingDir, 0700); err != nil {
		return nil, err
	}
	resp.StagingDir = stagingDir
	resp.RemoteRel = req.RemoteRel
	resp.RemotePath = filepath.Join(serverHome, filepath.FromSlash(req.RemoteRel))
	resp.StagedPath = filepath.Join(stagingDir, resp.RemotePath)

	if req.StaleStaged {
		stale := bytes.Repeat([]byte("Z"), len(readFileOrEmpty(resp.RemotePath)))
		if len(stale) == 0 {
			stale = []byte("stale")
		}
		if err := os.MkdirAll(filepath.Dir(resp.StagedPath), 0700); err != nil {
			return nil, err
		}
		if err := os.WriteFile(resp.StagedPath, stale, 0600); err != nil {
			return nil, err
		}
	}

	editorFlag, err := buildEditorScript(t, req, agentWorkDir, serverHome)
	if err != nil {
		return nil, err
	}
	resp.EditorPath = editorFlag

	if useBinaryPath(req) {
		return runBinaryE2E(t, d, req, resp, moduleRoot, cacheDir, serverHome, agentHome, agentWorkDir, editorFlag)
	}
	return runInProcessL2(t, d, req, resp, serverHome, agentHome, agentWorkDir, editorFlag)
}

// buildEditorScript generates the fake editor used by a leaf and returns the
// value for --editor ("" when the leaf supplies EditorFlag itself).
func buildEditorScript(t *testing.T, req *Request, agentWorkDir, serverHome string) (string, error) {
	t.Helper()
	if req.EditorFlag != "" {
		return req.EditorFlag, nil
	}
	if req.FakeTerminalEditor != "" {
		// An executable whose basename is a terminal editor, addressed by
		// absolute path so no PATH mutation is needed.
		binDir := filepath.Join(agentWorkDir, "bin")
		if err := os.MkdirAll(binDir, 0755); err != nil {
			return "", err
		}
		fake := filepath.Join(binDir, req.FakeTerminalEditor)
		marker := filepath.Join(agentWorkDir, "editor-ran")
		script := "#!/bin/sh\nprintf 'ran' > " + shellQuote(marker) + "\nexit 0\n"
		if err := os.WriteFile(fake, []byte(script), 0755); err != nil {
			return "", err
		}
		return fake, nil
	}

	script := filepath.Join(agentWorkDir, "editor.sh")
	var b strings.Builder
	b.WriteString("#!/bin/sh\n")
	// The staged path is the last argument, so --editor="script --flag" works.
	b.WriteString("last=\"\"\nfor a in \"$@\"; do last=\"$a\"; done\n")
	b.WriteString("counter=\"" + filepath.Join(agentWorkDir, "editor-count") + "\"\n")
	b.WriteString("n=0\n[ -f \"$counter\" ] && n=$(cat \"$counter\")\nn=$((n+1))\nprintf '%s' \"$n\" > \"$counter\"\n")

	// Edit the staged copy first: an abort (non-zero exit) must be observable
	// with changed staged content, so leaves prove nothing is uploaded.
	if !req.EditorNoop && (req.EditorWrite != "" || req.EditorWriteSecond != "") {
		first, err := writeEditorContent(t, agentWorkDir, "content-1", req.EditorWrite)
		if err != nil {
			return "", err
		}
		second := first
		if req.EditorWriteSecond != "" {
			second, err = writeEditorContent(t, agentWorkDir, "content-2", req.EditorWriteSecond)
			if err != nil {
				return "", err
			}
		}
		fmt.Fprintf(&b, "if [ \"$n\" -le 1 ]; then cp %s \"$last\"; else cp %s \"$last\"; fi\n", shellQuote(first), shellQuote(second))
	}

	for rel, content := range req.RemoteWrite {
		contentFile, err := writeEditorContent(t, agentWorkDir, "remote-"+strings.ReplaceAll(rel, "/", "_"), content)
		if err != nil {
			return "", err
		}
		full := filepath.Join(serverHome, filepath.FromSlash(rel))
		b.WriteString("mkdir -p " + shellQuote(filepath.Dir(full)) + "\n")
		b.WriteString("cp " + shellQuote(contentFile) + " " + shellQuote(full) + "\n")
	}
	for _, rel := range req.RemoteDelete {
		b.WriteString("rm -f " + shellQuote(filepath.Join(serverHome, filepath.FromSlash(rel))) + "\n")
	}
	if req.EditorDeleteStaged {
		b.WriteString("rm -f \"$last\"\n")
	}
	if req.EditorExitCode != 0 {
		fmt.Fprintf(&b, "exit %d\n", req.EditorExitCode)
	} else {
		b.WriteString("exit 0\n")
	}

	if err := os.WriteFile(script, []byte(b.String()), 0755); err != nil {
		return "", err
	}

	flag := script
	if req.EditorArgsSuffix != "" {
		flag = script + " " + req.EditorArgsSuffix
	}
	return flag, nil
}

func writeEditorContent(t *testing.T, dir, name, content string) (string, error) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return "", err
	}
	return path, nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func applyServerPreseed(t *testing.T, serverHome string, req *Request) error {
	t.Helper()
	for _, rel := range req.ServerPreseedDirs {
		full := filepath.Join(serverHome, filepath.FromSlash(rel))
		if err := os.MkdirAll(full, 0755); err != nil {
			return fmt.Errorf("mkdir preseed dir %s: %w", full, err)
		}
	}
	for rel, content := range req.ServerPreseedFiles {
		full := filepath.Join(serverHome, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			return fmt.Errorf("mkdir preseed parent %s: %w", filepath.Dir(full), err)
		}
		mode := os.FileMode(0644)
		if m, ok := req.ServerPreseedModes[rel]; ok {
			mode = m
		}
		if err := os.WriteFile(full, []byte(content), mode); err != nil {
			return fmt.Errorf("write preseed file %s: %w", full, err)
		}
		if err := os.Chmod(full, mode); err != nil {
			return fmt.Errorf("chmod preseed file %s: %w", full, err)
		}
	}
	for link, target := range req.ServerSymlinks {
		linkPath := filepath.Join(serverHome, filepath.FromSlash(link))
		if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
			return fmt.Errorf("mkdir symlink parent %s: %w", filepath.Dir(linkPath), err)
		}
		if err := os.Symlink(filepath.Join(serverHome, filepath.FromSlash(target)), linkPath); err != nil {
			return fmt.Errorf("symlink %s -> %s: %w", linkPath, target, err)
		}
	}
	return nil
}

// withEditArgs returns the argv for a run: server/token, the leaf args, an
// injected --work-dir when the leaf does not provide one, and (when
// injectEditor) an injected --editor for the generated editor script.
// injectRemember adds --remember-flags; the second run of a leaf must not
// re-save flags, otherwise it would overwrite the memory it is testing.
func withEditArgs(req *Request, args []string, serverURL, stagingDir, editorFlag string, injectEditor, injectRemember bool) []string {
	argv := []string{"--server", serverURL, "--token", req.Token}
	argv = append(argv, args...)
	if !argsContain(args, "--work-dir") && !argsContainPrefix(args, "--work-dir=") {
		argv = append(argv, "--work-dir", stagingDir)
	}
	if injectEditor && editorFlag != "" && !argsContain(args, "--editor") && !argsContainPrefix(args, "--editor=") {
		argv = append(argv, "--editor", editorFlag)
	}
	if injectRemember && req.RememberFlags && !argsContain(args, "--remember-flags") {
		argv = append(argv, "--remember-flags")
	}
	return argv
}

func argsContain(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}

func argsContainPrefix(args []string, prefix string) bool {
	for _, a := range args {
		if strings.HasPrefix(a, prefix) {
			return true
		}
	}
	return false
}

func runInProcessL2(t *testing.T, d *session.Doctest, req *Request, resp *Response, serverHome, agentHome, agentWorkDir, editorFlag string) (*Response, error) {
	t.Helper()

	mux := http.NewServeMux()
	fileupload.RegisterAPIForHome(mux, serverHome)
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen in-process edit server: %w", err)
	}
	serverPort := ln.Addr().(*net.TCPAddr).Port
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	serverURL := req.Server
	if serverURL == "" {
		serverURL = fmt.Sprintf("http://127.0.0.1:%d", serverPort)
	}
	resp.ServerURL = serverURL

	if err := waitHTTPReady(fmt.Sprintf("http://127.0.0.1:%d/ping", serverPort), 10*time.Second); err != nil {
		return nil, err
	}
	if err := verifyServerHome(t, serverURL, req.Token, serverHome); err != nil {
		return nil, err
	}
	t.Logf("L2 in-process edit server on %s home=%s", serverURL, serverHome)

	firstArgs := withEditArgs(req, req.Args, serverURL, resp.StagingDir, editorFlag, true, true)
	exitCode, stdout, stderr, err := runAgentInProcess(t, firstArgs, agentHome)
	if err != nil {
		return nil, err
	}
	resp.ExitCode, resp.Stdout, resp.Stderr = exitCode, stdout, stderr
	resp.Combined = strings.TrimSpace(stdout + "\n" + stderr)

	if len(req.SecondArgs) > 0 {
		secondArgs := withEditArgs(req, req.SecondArgs, serverURL, resp.StagingDir, editorFlag, false, false)
		exitCode, stdout, stderr, err := runAgentInProcess(t, secondArgs, agentHome)
		if err != nil {
			return nil, err
		}
		resp.SecondExitCode, resp.SecondStdout, resp.SecondStderr = exitCode, stdout, stderr
		resp.SecondCombined = strings.TrimSpace(stdout + "\n" + stderr)
	}

	resp.EditorRan = fileExists(filepath.Join(agentWorkDir, "editor-ran"))
	return resp, nil
}

// runAgentInProcess runs the CLI in-process with injected writers and an
// isolated config home (no process env/stdio/cwd mutation).
func runAgentInProcess(t *testing.T, argv []string, agentHome string) (int, string, string, error) {
	t.Helper()
	agentcliInProcessMu.Lock()
	defer agentcliInProcessMu.Unlock()

	testhooks.SetHomeOverride(agentHome)
	defer testhooks.ResetInProcessOverrides()

	var stdoutBuf, stderrBuf bytes.Buffer
	runErr := agentcli.RunWithWriters(agentcli.RemoteProfile(), argv, &stdoutBuf, &stderrBuf)

	stdout := stdoutBuf.String()
	stderr := stderrBuf.String()
	exitCode := 0
	if runErr != nil {
		stderr += fmt.Sprintf("Error: %v\n", runErr)
		exitCode = 1
	}
	return exitCode, stdout, stderr, nil
}

func runBinaryE2E(t *testing.T, d *session.Doctest, req *Request, resp *Response, moduleRoot, cacheDir, serverHome, agentHome, agentWorkDir, editorFlag string) (*Response, error) {
	t.Helper()

	serverBin, agentBin := buildSessionBinariesOnce(t, moduleRoot, cacheDir)

	credDir := filepath.Join(serverHome, ".ai-critic")
	if err := os.MkdirAll(credDir, 0755); err != nil {
		return nil, err
	}
	credFile := filepath.Join(credDir, "server-credentials")
	if err := os.WriteFile(credFile, []byte(req.Token+"\n"), 0600); err != nil {
		return nil, fmt.Errorf("write credentials: %w", err)
	}

	remoteConfigPath := filepath.Join(agentHome, ".ai-critic", "remote-agent-config.json")
	if err := os.MkdirAll(filepath.Dir(remoteConfigPath), 0755); err != nil {
		return nil, err
	}

	portBase := portBaseFromTestName(t.Name())
	serverPort := pickFreePort(portBase)

	serverURL := req.Server
	if serverURL == "" {
		serverURL = fmt.Sprintf("http://localhost:%d", serverPort)
	}
	resp.ServerURL = serverURL
	normalizedServer := strings.TrimRight(strings.TrimSpace(serverURL), "/")

	if err := writeRemoteAgentConfig(remoteConfigPath, normalizedServer, req.Token); err != nil {
		return nil, err
	}

	killPort(serverPort)

	// --no-event-bus-publish keeps the smoke server independent of any ai-critic
	// instance already running on this machine (its loopback publish port).
	serverCmd := exec.Command(serverBin, "--port", strconv.Itoa(serverPort),
		"--credentials-file", credFile, "--no-event-bus-publish")
	serverCmd.Dir = serverHome
	serverCmd.Env = stripEnvPrefix(os.Environ(), "HOME=")
	serverCmd.Env = stripEnvPrefix(serverCmd.Env, lib.EnvAI_CRITIC_HOME+"=")
	serverCmd.Env = append(serverCmd.Env, "HOME="+serverHome, "AI_CRITIC_NO_OPEN_BROWSER=1")
	if err := serverCmd.Start(); err != nil {
		return nil, fmt.Errorf("start server: %w", err)
	}
	t.Cleanup(func() {
		if serverCmd.Process != nil {
			serverCmd.Process.Signal(syscall.SIGTERM)
			time.Sleep(150 * time.Millisecond)
			serverCmd.Process.Kill()
		}
	})

	if err := waitHTTPReady(fmt.Sprintf("http://127.0.0.1:%d/ping", serverPort), 30*time.Second); err != nil {
		return nil, err
	}
	if err := verifyServerHome(t, normalizedServer, req.Token, serverHome); err != nil {
		return nil, err
	}

	agentEnv := stripEnvPrefix(os.Environ(), "HOME=")
	agentEnv = append(agentEnv, "HOME="+agentHome)

	firstArgs := withEditArgs(req, req.Args, serverURL, resp.StagingDir, editorFlag, true, true)
	exitCode, stdout, stderr, err := runAgentBinary(agentBin, firstArgs, agentEnv, agentWorkDir)
	if err != nil {
		return nil, err
	}
	resp.ExitCode, resp.Stdout, resp.Stderr = exitCode, stdout, stderr
	resp.Combined = strings.TrimSpace(stdout + "\n" + stderr)

	if len(req.SecondArgs) > 0 {
		secondArgs := withEditArgs(req, req.SecondArgs, serverURL, resp.StagingDir, editorFlag, false, false)
		exitCode, stdout, stderr, err := runAgentBinary(agentBin, secondArgs, agentEnv, agentWorkDir)
		if err != nil {
			return nil, err
		}
		resp.SecondExitCode, resp.SecondStdout, resp.SecondStderr = exitCode, stdout, stderr
		resp.SecondCombined = strings.TrimSpace(stdout + "\n" + stderr)
	}

	return resp, nil
}

func runAgentBinary(bin string, argv, env []string, dir string) (int, string, string, error) {
	cmd := exec.Command(bin, argv...)
	cmd.Env = env
	cmd.Dir = dir
	cmd.Stdin = nil
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	runErr := cmd.Run()
	exitCode := 0
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return 0, "", "", runErr
		}
	}
	return exitCode, outBuf.String(), errBuf.String(), nil
}

func readFileOrEmpty(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

type remoteAgentConfigFile struct {
	Default string            `json:"default,omitempty"`
	Domains []domainConfigRow `json:"domains"`
}

type domainConfigRow struct {
	Server string `json:"server"`
	Token  string `json:"token,omitempty"`
}

func writeRemoteAgentConfig(path, server, token string) error {
	cfg := remoteAgentConfigFile{
		Default: server,
		Domains: []domainConfigRow{{Server: server, Token: token}},
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func portBaseFromTestName(name string) int {
	hash := 0
	for _, c := range name {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return 31000 + (hash % 1000)
}

func pickFreePort(base int) int {
	for port := base; port < base+200; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			ln.Close()
			return port
		}
	}
	panic(fmt.Sprintf("no free port near %d", base))
}

func killPort(port int) {
	out, err := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", port)).Output()
	if err != nil {
		return
	}
	for _, pidStr := range strings.Fields(strings.TrimSpace(string(out))) {
		_ = exec.Command("kill", "-9", pidStr).Run()
	}
}

func stripEnvPrefix(env []string, prefix string) []string {
	out := make([]string, 0, len(env))
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			continue
		}
		out = append(out, e)
	}
	return out
}

func waitHTTPReady(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", url)
}

func normalizeAbsPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	eval, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs, nil
	}
	return eval, nil
}

func verifyServerHome(t *testing.T, serverURL, token, wantHome string) error {
	want, err := normalizeAbsPath(wantHome)
	if err != nil {
		return fmt.Errorf("resolve harness serverHome: %w", err)
	}
	homeURL := strings.TrimRight(strings.TrimSpace(serverURL), "/") + "/api/files/home"
	req, err := http.NewRequest(http.MethodGet, homeURL, nil)
	if err != nil {
		return fmt.Errorf("build verify-home request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("verify server HOME: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read verify-home response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("verify server HOME status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var home struct {
		Home string `json:"home"`
	}
	if err := json.Unmarshal(data, &home); err != nil {
		return fmt.Errorf("decode home response: %w", err)
	}
	got, err := normalizeAbsPath(home.Home)
	if err != nil {
		return fmt.Errorf("resolve server-reported HOME %q: %w", home.Home, err)
	}
	if got != want {
		return fmt.Errorf(
			"server HOME mismatch: server reports %q (normalized %q) but harness serverHome is %q (normalized %q)",
			home.Home, got, wantHome, want,
		)
	}
	t.Logf("verified server HOME=%s", got)
	return nil
}
```

