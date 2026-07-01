// Package buildfixtest provides shared helpers for terminal-refactor-build-fix doctests.
package buildfixtest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/xhd2015/ai-critic/server/terminal"
	"github.com/xhd2015/dot-pkgs/go-pkgs/shell/ptywrap"
	ptyclient "github.com/xhd2015/dot-pkgs/go-pkgs/shell/ptywrap/client"
	aicriticclient "github.com/xhd2015/ai-critic/client"
)

// Request is the doctest harness request for build-fix regression tests.
type Request struct {
	Phase string

	BuildTarget string

	ShellQuoteInput  string
	ShellQuoteInputs []string

	KeepAlivePort       int
	KeepAliveBinPath    string
	KeepAliveServerArgs []string

	WSExecArgv       []string
	WSStdoutPayload  string
	WSExitCode       int
	WSErrorMessage   string
	WSDialHTTPStatus int
	WSDialHTTPBody   string

	// WSAttachSilent holds the fake /api/terminal server open without ever
	// sending a session_id message, forcing the client's read-deadline path.
	WSAttachSilent bool

	// AttachKnownSessionID is the client-supplied session ID for attach-mode
	// leaves (mirrors `terminal attach <id>`).
	AttachKnownSessionID string
}

// Response is the doctest harness response.
type Response struct {
	BuildExitCode int
	BuildOutput   string

	ShellQuoteOutput  string
	ShellQuoteOutputs map[string]string
	ShellRoundTripOK  bool

	KeepAliveScript string
	KeepAliveShNOK  bool

	WSExitCode int
	WSStdout   string
	WSError    string

	// Attach outcome for the silent-server /api/terminal scenario.
	AttachPaniced   bool
	AttachPanic     string
	AttachErr       string
	AttachSessionID string
	// AttachKnownSessionID echoes the client-supplied ID for attach-mode leaves.
	AttachKnownSessionID string

	// RegisterAPI outcome for the server-boot route-registration smoke.
	RegisterPaniced bool
	RegisterPanic   string
}

// Run executes a build-fix regression phase.
func Run(t *testing.T, req *Request) (*Response, error) {
	switch req.Phase {
	case "remote-agent-build":
		return runCompile(t, "./cmd/remote-agent")
	case "server-build":
		return runCompile(t, ".")
	case "shell-quote-simple":
		return runShellQuoteSimple(t, req)
	case "shell-quote-special":
		return runShellQuoteSpecial(t, req)
	case "keep-alive-script":
		return runKeepAliveScript(t, req)
	case "ws-exec-exit-code":
		return runWSExecExitCode(t, req)
	case "ws-exec-error-message":
		return runWSExecErrorMessage(t, req)
	case "ws-dial-http-error":
		return runWSDialHTTPError(t, req)
	case "ws-attach-no-session-id":
		return runWSAttachNoSessionID(t, req)
	case "ws-attach-existing-session":
		return runWSAttachExistingSession(t, req)
	case "server-api-register":
		return runServerAPIRegister(t, req)
	default:
		return nil, fmt.Errorf("unknown phase %q", req.Phase)
	}
}

func runCompile(t *testing.T, pkg string) (*Response, error) {
	t.Helper()
	exitCode, output, err := CompilePackage(t, pkg)
	resp := &Response{
		BuildExitCode: exitCode,
		BuildOutput:   output,
	}
	if exitCode != 0 {
		return resp, fmt.Errorf("go build %s: exit %d\n%s", pkg, exitCode, output)
	}
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func runShellQuoteSimple(t *testing.T, req *Request) (*Response, error) {
	input := req.ShellQuoteInput
	if input == "" {
		input = "/tmp/ai-critic"
	}
	quoted := ptywrap.ShellQuote(input)
	ok, err := ShellQuoteRoundTrip(t, quoted, input)
	if err != nil {
		return nil, err
	}
	return &Response{
		ShellQuoteOutput: quoted,
		ShellRoundTripOK: ok,
	}, nil
}

func runShellQuoteSpecial(t *testing.T, req *Request) (*Response, error) {
	inputs := req.ShellQuoteInputs
	if len(inputs) == 0 {
		inputs = []string{"arg with spaces", "it's"}
	}
	outputs := make(map[string]string, len(inputs))
	for _, input := range inputs {
		quoted := ptywrap.ShellQuote(input)
		ok, err := ShellQuoteRoundTrip(t, quoted, input)
		if err != nil {
			return nil, err
		}
		if !ok {
			return &Response{ShellQuoteOutputs: outputs}, fmt.Errorf("shell round-trip failed for %q (quoted %q)", input, quoted)
		}
		if err := AssertNoShellInjection(t, quoted, input); err != nil {
			return &Response{ShellQuoteOutputs: outputs}, err
		}
		outputs[input] = quoted
	}
	return &Response{ShellQuoteOutputs: outputs}, nil
}

func runKeepAliveScript(t *testing.T, req *Request) (*Response, error) {
	binPath := req.KeepAliveBinPath
	if binPath == "" {
		binPath = filepath.Join(t.TempDir(), "ai critic", "ai-critic")
	}
	port := req.KeepAlivePort
	if port <= 0 {
		port = 14099
	}
	args := req.KeepAliveServerArgs
	if len(args) == 0 {
		args = []string{"--config", filepath.Join(t.TempDir(), "my config", "settings.json")}
	}

	script, err := OutputKeepAliveScript(port, args, binPath)
	if err != nil {
		return nil, err
	}
	if err := AssertKeepAliveScriptQuotesPaths(t, script, binPath, args); err != nil {
		return &Response{KeepAliveScript: script}, err
	}
	ok, err := ShellScriptSyntaxOK(script)
	if err != nil {
		return &Response{KeepAliveScript: script}, err
	}
	if !ok {
		return &Response{KeepAliveScript: script}, fmt.Errorf("sh -n syntax check failed")
	}
	return &Response{
		KeepAliveScript: script,
		KeepAliveShNOK:  true,
	}, nil
}

func runWSExecExitCode(t *testing.T, req *Request) (*Response, error) {
	argv := req.WSExecArgv
	if len(argv) == 0 {
		argv = []string{"echo", "hi"}
	}
	payload := req.WSStdoutPayload
	if payload == "" {
		payload = "hello from remote\n"
	}
	exitCode := req.WSExitCode
	if exitCode == 0 {
		exitCode = 42
	}

	srv, baseURL := startFakeExecWSServer(t, func(conn *websocket.Conn) error {
		if err := readWSExecRequest(conn); err != nil {
			return err
		}
		drainWSControlMessages(conn)
		if err := conn.WriteMessage(websocket.BinaryMessage, []byte(payload)); err != nil {
			return err
		}
		msg, err := json.Marshal(map[string]interface{}{
			"type": "exit",
			"code": exitCode,
		})
		if err != nil {
			return err
		}
		return conn.WriteMessage(websocket.TextMessage, msg)
	})
	defer srv.Close()

	var stdout bytes.Buffer
	code, err := runRemoteExecClient(t, baseURL, ptyclient.ExecOptions{
		Argv:         argv,
		Stdout:       &stdout,
		Stdin:        strings.NewReader(""),
		SkipTTYCheck: true,
	})
	if err != nil {
		return &Response{WSStdout: stdout.String()}, err
	}
	return &Response{
		WSExitCode: code,
		WSStdout:   stdout.String(),
	}, nil
}

func runWSExecErrorMessage(t *testing.T, req *Request) (*Response, error) {
	msg := req.WSErrorMessage
	if msg == "" {
		msg = "boom"
	}
	srv, baseURL := startFakeExecWSServer(t, func(conn *websocket.Conn) error {
		if err := readWSExecRequest(conn); err != nil {
			return err
		}
		drainWSControlMessages(conn)
		payload, err := json.Marshal(map[string]string{
			"type":    "error",
			"message": msg,
		})
		if err != nil {
			return err
		}
		return conn.WriteMessage(websocket.TextMessage, payload)
	})
	defer srv.Close()

	_, err := runRemoteExecClient(t, baseURL, ptyclient.ExecOptions{
		Argv:         []string{"true"},
		Stdout:       io.Discard,
		Stdin:        strings.NewReader(""),
		SkipTTYCheck: true,
	})
	if err == nil {
		return nil, fmt.Errorf("expected error containing %q", msg)
	}
	return &Response{WSError: err.Error()}, nil
}

func runWSDialHTTPError(t *testing.T, req *Request) (*Response, error) {
	status := req.WSDialHTTPStatus
	if status == 0 {
		status = http.StatusUnauthorized
	}
	body := req.WSDialHTTPBody
	if body == "" {
		body = `{"error":"unauthorized"}`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	_, err := runRemoteExecClient(t, srv.URL, ptyclient.ExecOptions{
		Argv:         []string{"true"},
		Stdout:       io.Discard,
		Stdin:        strings.NewReader(""),
		SkipTTYCheck: true,
	})
	if err == nil {
		return nil, fmt.Errorf("expected dial error for HTTP %d", status)
	}
	errText := err.Error()
	if !strings.Contains(errText, fmt.Sprintf("%d", status)) && !strings.Contains(errText, http.StatusText(status)) {
		return &Response{WSError: errText}, fmt.Errorf("error %q missing HTTP status %d", errText, status)
	}
	if !strings.Contains(errText, strings.TrimSpace(body)) {
		return &Response{WSError: errText}, fmt.Errorf("error %q missing body snippet %q", errText, body)
	}
	return &Response{WSError: errText}, nil
}

// runWSAttachNoSessionID dials a fake /api/terminal server that upgrades the
// WebSocket but never sends a session_id message. The client's readSessionID
// loop must surface a graceful timeout error instead of panicking with
// "repeated read on failed websocket connection".
func runWSAttachNoSessionID(t *testing.T, req *Request) (*Response, error) {
	hold := 15 * time.Second
	srv, baseURL := startFakeTerminalWSServer(t, func(conn *websocket.Conn) error {
		// Accept the upgrade, then hold the connection open without ever
		// sending a session_id so the client's read deadline is what fires.
		time.Sleep(hold)
		return nil
	})
	defer srv.Close()

	c := &ptyclient.Client{BaseURL: baseURL}

	type attachOutcome struct {
		result ptyclient.AttachResult
		err    error
	}
	outCh := make(chan attachOutcome, 1)
	panicCh := make(chan interface{}, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicCh <- r
			}
		}()
		res, err := ptyclient.AttachWithIO(c, ptyclient.ConnectOptions{
			Wait:         true,
			SkipTTYCheck: true,
		}, strings.NewReader(""), io.Discard, io.Discard)
		outCh <- attachOutcome{result: res, err: err}
	}()

	select {
	case p := <-panicCh:
		return &Response{AttachPaniced: true, AttachPanic: fmt.Sprint(p)}, nil
	case out := <-outCh:
		resp := &Response{AttachSessionID: out.result.SessionID}
		if out.err != nil {
			resp.AttachErr = out.err.Error()
		}
		return resp, nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("attach did not return within 30s")
	}
}

// runWSAttachExistingSession reproduces the client/backend skew from the
// report: a stale pre-refactor daemon reattaches to an existing session by
// accepting the WebSocket and immediately serving it (writing a binary output
// frame) WITHOUT echoing a {"type":"session_id"} frame. The pre-refactor
// handler only emitted session_id on the create paths (s == nil), not on
// reattach. The new dot-pkgs client's readSessionID unconditionally demands a
// session_id frame, so it times out against such a daemon even though the
// client already supplied the SessionID it wants to attach to.
func runWSAttachExistingSession(t *testing.T, req *Request) (*Response, error) {
	knownSessionID := req.AttachKnownSessionID
	if knownSessionID == "" {
		knownSessionID = "session-5"
	}

	srv, baseURL := startFakeTerminalWSServer(t, func(conn *websocket.Conn) error {
		// Mimic pre-refactor reattach: do NOT send a session_id frame. Begin
		// serving immediately by writing a binary output frame (the session is
		// live and streaming), then hold the connection open.
		if err := conn.WriteMessage(websocket.BinaryMessage, []byte("$ \r\n")); err != nil {
			return err
		}
		time.Sleep(15 * time.Second)
		return nil
	})
	defer srv.Close()

	c := &ptyclient.Client{BaseURL: baseURL}

	type attachOutcome struct {
		result ptyclient.AttachResult
		err    error
	}
	outCh := make(chan attachOutcome, 1)
	panicCh := make(chan interface{}, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicCh <- r
			}
		}()
		res, err := ptyclient.AttachWithIO(c, ptyclient.ConnectOptions{
			SessionID:      knownSessionID,
			AttachSnapshot: true,
			Wait:           true,
			SkipTTYCheck:   true,
		}, strings.NewReader(""), io.Discard, io.Discard)
		outCh <- attachOutcome{result: res, err: err}
	}()

	select {
	case p := <-panicCh:
		return &Response{AttachPaniced: true, AttachPanic: fmt.Sprint(p), AttachSessionID: knownSessionID}, nil
	case out := <-outCh:
		resp := &Response{AttachSessionID: out.result.SessionID, AttachKnownSessionID: knownSessionID}
		if out.err != nil {
			resp.AttachErr = out.err.Error()
		}
		return resp, nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("attach did not return within 30s")
	}
}

// runServerAPIRegister is a server-boot smoke: terminal.RegisterAPI must
// register all terminal routes on a fresh ServeMux without panicking. The
// post-refactor adapter double-registers /api/terminal (ptywrap already
// registers it, then the SSH wrapper registers it again), which panics under
// Go 1.22+ ServeMux and prevents the server from booting.
func runServerAPIRegister(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	mux := http.NewServeMux()
	func() {
		defer func() {
			if r := recover(); r != nil {
				resp.RegisterPaniced = true
				resp.RegisterPanic = fmt.Sprint(r)
			}
		}()
		terminal.RegisterAPI(mux)
	}()
	return resp, nil
}

// startFakeTerminalWSServer serves /api/terminal (the ptywrap attach endpoint)
// and runs session against each upgraded connection.
func startFakeTerminalWSServer(t *testing.T, session func(conn *websocket.Conn) error) (*httptest.Server, string) {
	t.Helper()
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/terminal" {
			http.NotFound(w, r)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		if err := session(conn); err != nil {
			t.Errorf("fake terminal ws session: %v", err)
		}
	}))
	return srv, srv.URL
}

// ModuleRoot returns the ai-critic module root directory.
func ModuleRoot(t *testing.T) string {
	t.Helper()
	if root := os.Getenv("DOCTEST_ROOT"); root != "" {
		for dir := root; ; dir = filepath.Dir(dir) {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				return dir
			}
			if filepath.Dir(dir) == dir {
				break
			}
		}
	}
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
	return ""
}

// CompilePackage runs `go build -o /dev/null` for pkg relative to module root.
func CompilePackage(t *testing.T, pkg string) (exitCode int, output string, err error) {
	t.Helper()
	moduleRoot := ModuleRoot(t)
	cmd := exec.Command("go", "build", "-o", "/dev/null", pkg)
	cmd.Dir = moduleRoot
	out, runErr := cmd.CombinedOutput()
	output = string(out)
	if runErr == nil {
		return 0, output, nil
	}
	if exitErr, ok := runErr.(*exec.ExitError); ok {
		return exitErr.ExitCode(), output, nil
	}
	return 1, output, runErr
}

// ShellQuoteRoundTrip verifies quoted text round-trips through POSIX sh.
func ShellQuoteRoundTrip(t *testing.T, quoted, want string) (bool, error) {
	t.Helper()
	script := fmt.Sprintf("v=%s; printf '%%s' \"$v\"", quoted)
	cmd := exec.Command("sh", "-c", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("sh round-trip for %q (quoted %q): %v\n%s", want, quoted, err, out)
	}
	if string(out) != want {
		return false, fmt.Errorf("sh round-trip for %q got %q, want %q (quoted %q)", want, string(out), want, quoted)
	}
	return true, nil
}

// AssertNoShellInjection ensures adjacent unquoted text cannot alter the quoted value.
func AssertNoShellInjection(t *testing.T, quoted, want string) error {
	t.Helper()
	script := fmt.Sprintf("v=%sx; printf '%%s' \"$v\"", quoted)
	cmd := exec.Command("sh", "-c", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("injection probe for %q: %v\n%s", want, err, out)
	}
	got := string(out)
	if got != want+"x" {
		return fmt.Errorf("quoted %q allowed injection: sh produced %q, want %q", quoted, got, want+"x")
	}
	return nil
}

// ShellScriptSyntaxOK runs `sh -n` against script text.
func ShellScriptSyntaxOK(script string) (bool, error) {
	cmd := exec.Command("sh", "-n", "-c", script)
	if out, err := cmd.CombinedOutput(); err != nil {
		return false, fmt.Errorf("sh -n: %v\n%s", err, out)
	}
	return true, nil
}

// AssertKeepAliveScriptQuotesPaths checks generated script embeds shell-quoted paths.
func AssertKeepAliveScriptQuotesPaths(t *testing.T, script, binPath string, serverArgs []string) error {
	t.Helper()
	binQuoted := ptywrap.ShellQuote(binPath)
	if !strings.Contains(script, binQuoted) {
		return fmt.Errorf("keep-alive script missing quoted bin path %q\n%s", binQuoted, script)
	}
	for _, arg := range serverArgs {
		if strings.ContainsAny(arg, " \t") || strings.Contains(arg, "'") {
			quoted := ptywrap.ShellQuote(arg)
			if !strings.Contains(script, quoted) {
				return fmt.Errorf("keep-alive script missing quoted arg %q (%q)\n%s", arg, quoted, script)
			}
		}
	}
	return nil
}

// OutputKeepAliveScript renders the keep-alive shell script via run export helper.
func OutputKeepAliveScript(port int, serverArgs []string, binPath string) (string, error) {
	return outputKeepAliveScriptForTest(port, serverArgs, binPath)
}

func startFakeExecWSServer(t *testing.T, session func(conn *websocket.Conn) error) (*httptest.Server, string) {
	t.Helper()
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/exec/ws" {
			http.NotFound(w, r)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		if err := session(conn); err != nil {
			t.Errorf("fake exec ws session: %v", err)
		}
	}))
	return srv, srv.URL
}

func readWSExecRequest(conn *websocket.Conn) error {
	_, data, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("read exec request: %w", err)
	}
	var req aicriticclient.ExecRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("decode exec request: %w", err)
	}
	if len(req.Argv) == 0 {
		return fmt.Errorf("exec request argv is empty")
	}
	return nil
}

func drainWSControlMessages(conn *websocket.Conn) {
	_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if msgType != websocket.TextMessage {
			continue
		}
		var ctrl struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(data, &ctrl) == nil && ctrl.Type == "resize" {
			continue
		}
		return
	}
}

func runRemoteExecClient(t *testing.T, serverURL string, opts ptyclient.ExecOptions) (int, error) {
	t.Helper()
	c := &ptyclient.Client{
		BaseURL: serverURL,
	}
	return ptyclient.RunExec(c, opts)
}