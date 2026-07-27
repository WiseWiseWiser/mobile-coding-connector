# Scenario

**Feature**: shared agent `config` CLI flag contract (remote + local)

```
# Wave C: help / validation → in-process agentcli.Run (L2)
# --show (loadConfig) → binary + cmd.Env HOME (isolated; not e2e-labeled)
# leaf Setup sets Profile + config args + optional seed file
test harness -> agentcli.Run | remote-agent|local-agent config [...] -> help | JSON | error
```

## Preconditions

1. Doctest injects `DOCTEST_SESSION_ID` for binary session cache when `--show` leaves run.
2. In-process path uses a suite mutex (profile + stdout capture); no `Setenv`/`Chdir`.
3. Binary path: temp `HOME` via `cmd.Env` only; 4s kill timer; no server.
4. No `ai-critic-server` is required for this tree.

## Steps

1. Group/leaf `Setup` sets `Profile` and `Args` (and seed for `--show`).
2. `Run` chooses in-process vs binary (`needsIsolatedHOME` / `UseCLI`).
3. Leaf `Assert` checks exit code, help/JSON/error text, and absence of UI banners.

## Context

Implements REQUIREMENT-DESIGN-remote-agent-config-flags.md (classic TDD / RED first).
`--web` full browser E2E is intentionally out of scope; bare must not print
`Config UI running`.

```go
import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	if req.Args == nil {
		req.Args = []string{}
	}
	if d.DOCTEST_SESSION_ID == "" {
		t.Fatal("session id empty on session.Doctest")
	}
	return nil
}

// sampleRemoteConfig is a stable multi-domain seed for remote-agent --show leaves.
func sampleRemoteConfig() *AgentConfigFile {
	return &AgentConfigFile{
		Default: "https://prod.example.com",
		Domains: []DomainEntry{
			{Server: "https://prod.example.com", Token: "tok-prod-full"},
			{Server: "https://staging.example.com", Token: "tok-staging"},
		},
	}
}

// sampleLocalConfig is a stable seed for local-agent --show leaves.
func sampleLocalConfig() *AgentConfigFile {
	return &AgentConfigFile{
		Default: "http://localhost:23712",
		Domains: []DomainEntry{
			{Server: "http://localhost:23712", Token: "local-secret-token"},
		},
	}
}

// remoteOnlySentinel is written beside local config to prove isolation.
func remoteOnlySentinel() *AgentConfigFile {
	return &AgentConfigFile{
		Default: "https://should-not-be-read.example.com",
		Domains: []DomainEntry{
			{Server: "https://should-not-be-read.example.com", Token: "remote-only-token"},
		},
	}
}

func assertNotTimedOut(t *testing.T, resp *Response) {
	t.Helper()
	if resp.TimedOut {
		t.Fatalf("CLI timed out (likely still opens Config UI on bare path); stdout:\n%s\nstderr:\n%s",
			resp.Stdout, resp.Stderr)
	}
}

func assertExitZero(t *testing.T, resp *Response) {
	t.Helper()
	assertNotTimedOut(t, resp)
	if resp.ExitCode != 0 {
		t.Fatalf("expected exit 0, got %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
}

func assertExitNonZero(t *testing.T, resp *Response) {
	t.Helper()
	assertNotTimedOut(t, resp)
	if resp.ExitCode == 0 {
		t.Fatalf("expected non-zero exit; combined:\n%s", resp.Combined)
	}
}

func assertNoConfigUI(t *testing.T, resp *Response) {
	t.Helper()
	if strings.Contains(resp.Stdout, "Config UI running") || strings.Contains(resp.Stderr, "Config UI running") {
		t.Fatalf("must not start Config UI; combined:\n%s", resp.Combined)
	}
}

func assertHelpMentionsFlags(t *testing.T, stdout, cliName string) {
	t.Helper()
	if !strings.Contains(stdout, "Usage:") {
		t.Fatalf("expected Usage header; stdout:\n%s", stdout)
	}
	if !strings.Contains(stdout, cliName) {
		t.Fatalf("expected help to mention %q; stdout:\n%s", cliName, stdout)
	}
	if !strings.Contains(stdout, "config") {
		t.Fatalf("expected help to mention config; stdout:\n%s", stdout)
	}
	// Target help documents the new flags (at least --web and --show).
	lower := strings.ToLower(stdout)
	if !strings.Contains(lower, "--web") {
		t.Fatalf("help should mention --web; stdout:\n%s", stdout)
	}
	if !strings.Contains(lower, "--show") {
		t.Fatalf("help should mention --show; stdout:\n%s", stdout)
	}
}

func assertPrettyEmptyishConfigJSON(t *testing.T, stdout string) {
	t.Helper()
	trimmed := strings.TrimSpace(stdout)
	if trimmed == "" {
		t.Fatal("expected JSON on stdout, got empty")
	}
	if !strings.Contains(stdout, "\n") {
		t.Fatalf("expected pretty-printed JSON (multi-line); got: %q", stdout)
	}
	var cfg AgentConfigFile
	if err := json.Unmarshal([]byte(trimmed), &cfg); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if len(cfg.Domains) != 0 {
		t.Fatalf("expected empty domains, got %#v", cfg.Domains)
	}
	if cfg.Default != "" {
		t.Fatalf("expected empty default, got %q", cfg.Default)
	}
}

func assertPrettyConfigMatches(t *testing.T, stdout string, want *AgentConfigFile) {
	t.Helper()
	trimmed := strings.TrimSpace(stdout)
	if !strings.Contains(stdout, "\n") {
		t.Fatalf("expected pretty-printed JSON; got: %q", stdout)
	}
	var got AgentConfigFile
	if err := json.Unmarshal([]byte(trimmed), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if got.Default != want.Default {
		t.Fatalf("default: want %q got %q", want.Default, got.Default)
	}
	if len(got.Domains) != len(want.Domains) {
		t.Fatalf("domains len: want %d got %d\nstdout:\n%s", len(want.Domains), len(got.Domains), stdout)
	}
	for i := range want.Domains {
		if got.Domains[i].Server != want.Domains[i].Server {
			t.Fatalf("domains[%d].server: want %q got %q", i, want.Domains[i].Server, got.Domains[i].Server)
		}
		if got.Domains[i].Token != want.Domains[i].Token {
			t.Fatalf("domains[%d].token: want full token %q got %q (no redaction)", i, want.Domains[i].Token, got.Domains[i].Token)
		}
	}
	// Pretty form should match MarshalIndent("", "  ") spirit.
	reencoded, err := json.MarshalIndent(&got, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(reencoded)) != trimmed {
		// Allow trailing newline on CLI stdout.
		if strings.TrimSpace(string(reencoded)+"\n") != strings.TrimSpace(stdout) {
			t.Logf("note: pretty layout may differ slightly; structural match already checked")
		}
	}
}

func combinedLower(resp *Response) string {
	return strings.ToLower(resp.Combined)
}
```
