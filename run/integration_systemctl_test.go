package run

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func resetIntegrationHooks() {
	integrationOut = nil
	integrationErr = nil
	integrationGOOS = "linux" // tests assume Linux unless overridden
	integrationExecutable = os.Executable
	integrationGetuid = os.Getuid
	integrationReadFile = os.ReadFile
	integrationWriteFile = os.WriteFile
	integrationMkdirAll = os.MkdirAll
	integrationHome = "/root"
	integrationUnitPath = sysctlUnitPath
	integrationSystemctl = defaultSystemctl
	integrationJournalctl = func(args []string) error {
		panic("journalctl should not run in tests")
	}
	integrationListen = func(int) bool { return false }
	integrationPing = func(int) (int, string, error) { return 200, "pong", nil }
}

func TestGenerateSystemdUnit(t *testing.T) {
	got := generateSystemdUnit("/root/servers/ai-critic/ai-critic-server", "/root/servers/ai-critic", "/root", 23712)
	for _, want := range []string{
		"[Unit]",
		"Description=ai-critic-server",
		"WorkingDirectory=/root/servers/ai-critic",
		"Environment=HOME=/root",
		"ExecStart=/root/servers/ai-critic/ai-critic-server --port 23712",
		"Restart=always",
		"[Install]",
		"WantedBy=multi-user.target",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("unit missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "keep-alive") {
		t.Fatal("unit must not ExecStart keep-alive")
	}
}

func TestUnitsMatchNormalizesWhitespace(t *testing.T) {
	a := generateSystemdUnit("/bin/x", "/bin", "/root", 23712)
	b := a + "\n\n"
	if !unitsMatch(a, b) {
		t.Fatal("trailing newlines should match")
	}
	if unitsMatch(a, strings.Replace(a, "--port 23712", "--port 9", 1)) {
		t.Fatal("port change must not match")
	}
}

func TestIntegrationHelp(t *testing.T) {
	resetIntegrationHooks()
	t.Cleanup(resetIntegrationHooks)
	var out bytes.Buffer
	integrationOut = &out
	if err := Run([]string{"integration", "-h"}); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, "Usage: ai-critic integration") || !strings.Contains(s, "systemctl") {
		t.Fatalf("integration help:\n%s", s)
	}
}

func TestIntegrationSystemctlHelp(t *testing.T) {
	resetIntegrationHooks()
	t.Cleanup(resetIntegrationHooks)
	var out bytes.Buffer
	integrationOut = &out
	if err := Run([]string{"integration", "systemctl", "-h"}); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{"status", "start", "stop", "logs", "show-config", "--dry-run"} {
		if !strings.Contains(s, want) {
			t.Fatalf("systemctl help missing %q:\n%s", want, s)
		}
	}
}

func TestIntegrationSystemctlNonLinux(t *testing.T) {
	resetIntegrationHooks()
	t.Cleanup(resetIntegrationHooks)
	integrationGOOS = "darwin"
	err := Run([]string{"integration", "systemctl", "status"})
	if err == nil || !strings.Contains(err.Error(), "Linux-only") {
		t.Fatalf("err %v", err)
	}
}

func TestIntegrationSystemctlStartDryRunMissingUnit(t *testing.T) {
	resetIntegrationHooks()
	t.Cleanup(resetIntegrationHooks)
	dir := t.TempDir()
	integrationUnitPath = filepath.Join(dir, "ai-critic-server.service")
	integrationExecutable = func() (string, error) {
		return "/root/servers/ai-critic/ai-critic-server", nil
	}
	integrationSystemctl = func(args ...string) (string, error) {
		t.Fatalf("systemctl should not run on dry-run: %v", args)
		return "", nil
	}
	var out bytes.Buffer
	integrationOut = &out
	if err := Run([]string{"integration", "systemctl", "start", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{
		"would write " + integrationUnitPath,
		"would systemctl daemon-reload",
		"would systemctl enable ai-critic-server",
		"would systemctl restart ai-critic-server",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("dry-run missing %q:\n%s", want, s)
		}
	}
}

func TestIntegrationSystemctlStartDryRunMatchSkipsWrite(t *testing.T) {
	resetIntegrationHooks()
	t.Cleanup(resetIntegrationHooks)
	dir := t.TempDir()
	integrationUnitPath = filepath.Join(dir, "ai-critic-server.service")
	bin := "/root/servers/ai-critic/ai-critic-server"
	integrationExecutable = func() (string, error) { return bin, nil }
	unit := generateSystemdUnit(bin, filepath.Dir(bin), "/root", 23712)
	if err := os.WriteFile(integrationUnitPath, []byte(unit), 0o644); err != nil {
		t.Fatal(err)
	}
	integrationSystemctl = func(args ...string) (string, error) {
		t.Fatalf("systemctl should not run on dry-run: %v", args)
		return "", nil
	}
	var out bytes.Buffer
	integrationOut = &out
	if err := Run([]string{"integration", "systemctl", "start", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if strings.Contains(s, "would write") || strings.Contains(s, "daemon-reload") {
		t.Fatalf("match should not rewrite:\n%s", s)
	}
	if !strings.Contains(s, "would systemctl enable") || !strings.Contains(s, "would systemctl restart") {
		t.Fatalf("dry-run:\n%s", s)
	}
}

func TestIntegrationSystemctlStartNeedsRoot(t *testing.T) {
	resetIntegrationHooks()
	t.Cleanup(resetIntegrationHooks)
	integrationGetuid = func() int { return 1 }
	err := Run([]string{"integration", "systemctl", "start"})
	if err == nil || !strings.Contains(err.Error(), "need root") {
		t.Fatalf("err %v", err)
	}
}

func TestIntegrationSystemctlShowConfig(t *testing.T) {
	resetIntegrationHooks()
	t.Cleanup(resetIntegrationHooks)
	dir := t.TempDir()
	integrationUnitPath = filepath.Join(dir, "ai-critic-server.service")
	integrationExecutable = func() (string, error) {
		return "/root/servers/ai-critic/ai-critic-server", nil
	}
	var out bytes.Buffer
	integrationOut = &out
	if err := Run([]string{"integration", "systemctl", "show-config"}); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{
		"unit:",
		"binary:     /root/servers/ai-critic/ai-critic-server",
		"on-disk:    missing",
		"# unit",
		"ExecStart=/root/servers/ai-critic/ai-critic-server --port 23712",
		"# day-to-day",
		"integration systemctl status",
		"integration systemctl logs",
		"journalctl -u ai-critic-server -n 100 -f --no-pager",
		"curl -sS http://127.0.0.1:23712/ping",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("show-config missing %q:\n%s", want, s)
		}
	}
}

func TestIntegrationSystemctlStartWritesAndRestarts(t *testing.T) {
	resetIntegrationHooks()
	t.Cleanup(resetIntegrationHooks)
	dir := t.TempDir()
	integrationUnitPath = filepath.Join(dir, "ai-critic-server.service")
	integrationGetuid = func() int { return 0 }
	integrationExecutable = func() (string, error) {
		return "/root/servers/ai-critic/ai-critic-server", nil
	}
	var calls []string
	integrationSystemctl = func(args ...string) (string, error) {
		calls = append(calls, strings.Join(args, " "))
		switch args[0] {
		case "show":
			if strings.Contains(strings.Join(args, " "), "ActiveState") {
				return "active\n", nil
			}
			if strings.Contains(strings.Join(args, " "), "MainPID") {
				return "74817\n", nil
			}
		}
		return "", nil
	}
	var out bytes.Buffer
	integrationOut = &out
	if err := Run([]string{"integration", "systemctl", "start", "--port", "23712"}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(integrationUnitPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "--port 23712") {
		t.Fatalf("unit:\n%s", body)
	}
	joined := strings.Join(calls, " | ")
	if !strings.Contains(joined, "daemon-reload") || !strings.Contains(joined, "enable") || !strings.Contains(joined, "restart") {
		t.Fatalf("systemctl calls: %v", calls)
	}
	s := out.String()
	if !strings.Contains(s, "unit:     created") || !strings.Contains(s, "enable:   ok") || !strings.Contains(s, "pid=74817") || !strings.Contains(s, "ping:     200 pong") {
		t.Fatalf("start out:\n%s", s)
	}
}

func TestIntegrationSystemctlStartWarnsKeepAlive(t *testing.T) {
	resetIntegrationHooks()
	t.Cleanup(resetIntegrationHooks)
	dir := t.TempDir()
	integrationUnitPath = filepath.Join(dir, "ai-critic-server.service")
	integrationGetuid = func() int { return 0 }
	integrationExecutable = func() (string, error) { return "/opt/ai-critic-server", nil }
	integrationListen = func(port int) bool { return port == 23312 }
	integrationSystemctl = func(args ...string) (string, error) {
		if args[0] == "show" {
			return "inactive\n", nil
		}
		return "", nil
	}
	var errBuf bytes.Buffer
	var out bytes.Buffer
	integrationOut = &out
	integrationErr = &errBuf
	if err := Run([]string{"integration", "systemctl", "start"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errBuf.String(), "warning: keep-alive looks running") {
		t.Fatalf("stderr: %s", errBuf.String())
	}
}

func TestIntegrationUnknownTarget(t *testing.T) {
	resetIntegrationHooks()
	t.Cleanup(resetIntegrationHooks)
	err := Run([]string{"integration", "launchd"})
	if err == nil || !strings.Contains(err.Error(), "unknown integration target") {
		t.Fatalf("err %v", err)
	}
}

func TestIsManagedServerChildIntegration(t *testing.T) {
	orig := os.Args
	t.Cleanup(func() { os.Args = orig })
	os.Args = []string{"ai-critic-server", "integration", "systemctl", "start"}
	if isManagedServerChild() {
		t.Fatal("integration subcommand is not a managed server child")
	}
}

func TestIntegrationSystemctlLogsDryRun(t *testing.T) {
	resetIntegrationHooks()
	t.Cleanup(resetIntegrationHooks)
	var out bytes.Buffer
	integrationOut = &out
	if err := Run([]string{"integration", "systemctl", "logs", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if s != "would journalctl -u ai-critic-server -n 100 --no-pager -f\n" {
		t.Fatalf("logs dry-run: %q", s)
	}
}

func TestIntegrationSystemctlLogsNoFollowLines(t *testing.T) {
	resetIntegrationHooks()
	t.Cleanup(resetIntegrationHooks)
	var out bytes.Buffer
	integrationOut = &out
	if err := Run([]string{"integration", "systemctl", "logs", "--dry-run", "--no-follow", "--lines", "20"}); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if s != "would journalctl -u ai-critic-server -n 20 --no-pager\n" {
		t.Fatalf("logs dry-run: %q", s)
	}
}

func TestJournalctlArgs(t *testing.T) {
	got := strings.Join(journalctlArgs(0, true), " ")
	if got != "-u ai-critic-server -n 100 --no-pager -f" {
		t.Fatalf("default args: %q", got)
	}
}
