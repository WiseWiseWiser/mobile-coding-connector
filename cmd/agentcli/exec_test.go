package agentcli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/xhd2015/ai-critic/client"
	serverexec "github.com/xhd2015/ai-critic/server/exec"
)

const execTestToken = "exec-test-token"

// execArgs builds a full CLI invocation whose argv after 'exec' is exactly
// args. --server/--token precede the subcommand, and the root parser stops at
// the first positional, so everything after 'exec' is left untouched.
func execArgs(serverURL string, args []string) []string {
	return append([]string{"--server", serverURL, "--token", execTestToken, "exec"}, args...)
}

// execFakeServer records the argv of every /api/exec request and replies with
// a canned NDJSON stream, so tests assert the exact argv that reaches the wire.
type execFakeServer struct {
	server *httptest.Server

	mu      sync.Mutex
	reqArgv [][]string
}

func newExecFakeServer(t *testing.T, stdout string) *execFakeServer {
	t.Helper()
	fake := &execFakeServer{}
	fake.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/exec" {
			t.Errorf("unexpected request path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		var req client.ExecRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode exec request: %v", err)
			return
		}
		fake.mu.Lock()
		fake.reqArgv = append(fake.reqArgv, req.Argv)
		fake.mu.Unlock()

		w.Header().Set("Content-Type", "application/x-ndjson")
		if stdout != "" {
			writeTestExecEvent(t, w, client.ExecEvent{Type: "stdout", Data: stdout})
		}
		writeTestExecEvent(t, w, client.ExecEvent{Type: "exit"})
	}))
	t.Cleanup(fake.server.Close)
	return fake
}

func writeTestExecEvent(t *testing.T, w http.ResponseWriter, ev client.ExecEvent) {
	t.Helper()
	data, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal exec event: %v", err)
	}
	if _, err := fmt.Fprintf(w, "%s\n", data); err != nil {
		t.Errorf("write exec event: %v", err)
	}
}

func (f *execFakeServer) argvs() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.reqArgv)
}

// runExecCLI runs the CLI in-process with process stdio redirected to buffers.
// RunWithWriters swaps os.Stdout for a pipe, which both captures runExec's
// direct os.Stdout writes and keeps term.IsTerminal false, so the
// non-interactive (NDJSON) path is deterministic under any test runner.
func runExecCLI(t *testing.T, serverURL string, args []string) (stdout, stderr string, err error) {
	t.Helper()
	var out, errOut bytes.Buffer
	err = RunWithWriters(RemoteProfile(), execArgs(serverURL, args), &out, &errOut)
	return out.String(), errOut.String(), err
}

// TestRunExecForwardsArgvWithoutLeadingSeparator pins the client-side
// separator contract: one leading '--' is consumed, everything else is
// forwarded verbatim (an internal '--' belongs to the remote binary).
func TestRunExecForwardsArgvWithoutLeadingSeparator(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "separator before binary",
			args: []string{"--", "sh", "-c", "echo yes"},
			want: []string{"sh", "-c", "echo yes"},
		},
		{
			name: "no separator unchanged",
			args: []string{"sh", "-c", "echo yes"},
			want: []string{"sh", "-c", "echo yes"},
		},
		{
			name: "internal separator preserved",
			args: []string{"echo", "--", "hi"},
			want: []string{"echo", "--", "hi"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := newExecFakeServer(t, "yes\n")
			stdout, stderr, err := runExecCLI(t, fake.server.URL, tc.args)
			if err != nil {
				t.Fatalf("exec %v: %v (stderr: %s)", tc.args, err, stderr)
			}
			got := fake.argvs()
			if len(got) != 1 {
				t.Fatalf("exec requests = %v, want exactly 1", got)
			}
			if !slices.Equal(got[0], tc.want) {
				t.Fatalf("wire argv = %v, want %v", got[0], tc.want)
			}
			if stdout != "yes\n" {
				t.Fatalf("stdout = %q, want %q", stdout, "yes\n")
			}
		})
	}
}

// TestRunExecSeparatorWithoutBinaryFailsLocally covers 'exec --': the
// separator leaves no binary, so the CLI must fail before any HTTP call
// instead of asking the server to resolve a binary named "--".
func TestRunExecSeparatorWithoutBinaryFailsLocally(t *testing.T) {
	fake := newExecFakeServer(t, "")
	stdout, stderr, err := runExecCLI(t, fake.server.URL, []string{"--"})
	if err == nil {
		t.Fatal("exec -- should fail without a binary")
	}
	if !strings.Contains(err.Error(), "exec requires <BINARY> [ARGS...]") {
		t.Fatalf("error = %v, want the exec requires <BINARY> hint", err)
	}
	if got := fake.argvs(); len(got) != 0 {
		t.Fatalf("exec -- must not reach the server; requests = %v", got)
	}
	if stdout != "" || stderr != "" {
		t.Fatalf("stdout = %q, stderr = %q, want both empty", stdout, stderr)
	}
}

// TestRunExecHelpAfterSeparatorPrintsClientHelp covers 'exec -- --help': the
// separator is dropped first, so the only client-side token stays reachable.
func TestRunExecHelpAfterSeparatorPrintsClientHelp(t *testing.T) {
	fake := newExecFakeServer(t, "")
	stdout, stderr, err := runExecCLI(t, fake.server.URL, []string{"--", "--help"})
	if err != nil {
		t.Fatalf("exec -- --help: %v (stderr: %s)", err, stderr)
	}
	if !strings.Contains(stdout, "Usage: remote-agent exec <BINARY> [ARGS...]") {
		t.Fatalf("stdout = %q, want the exec usage", stdout)
	}
	if !strings.Contains(stdout, "A leading '--' separator is accepted and ignored") {
		t.Fatalf("help should document the accepted separator:\n%s", stdout)
	}
	if got := fake.argvs(); len(got) != 0 {
		t.Fatalf("help must not reach the server; requests = %v", got)
	}
}

// TestRunExecSeparatorEndToEnd runs the reported command against the real
// /api/exec handler (in-process), covering the full client path: root parse ->
// separator strip -> NDJSON stream -> remote exit code.
func TestRunExecSeparatorEndToEnd(t *testing.T) {
	mux := http.NewServeMux()
	serverexec.RegisterAPI(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	// Remote exit codes stay 0 here: runExec calls os.Exit on non-zero, which
	// would tear down the test process.
	t.Run("separator dropped then remote sh runs", func(t *testing.T) {
		stdout, stderr, err := runExecCLI(t, server.URL, []string{"--", "sh", "-c", "echo yes"})
		if err != nil {
			t.Fatalf("exec -- sh -c 'echo yes': %v (stderr: %s)", err, stderr)
		}
		if !strings.Contains(stdout, "yes") {
			t.Fatalf("stdout = %q, want the remote 'yes'", stdout)
		}
	})

	t.Run("internal separator reaches remote", func(t *testing.T) {
		stdout, stderr, err := runExecCLI(t, server.URL, []string{"echo", "--", "hi"})
		if err != nil {
			t.Fatalf("exec echo -- hi: %v (stderr: %s)", err, stderr)
		}
		if !strings.Contains(stdout, "-- hi") {
			t.Fatalf("stdout = %q, want the remote '-- hi'", stdout)
		}
	})
}
