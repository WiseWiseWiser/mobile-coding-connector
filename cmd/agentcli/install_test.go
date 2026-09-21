package agentcli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/client"
)

func TestRunInstall_Help(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := RunWithWriters(RemoteProfile(), []string{"install", "--help"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("help: %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "Usage: remote-agent install <cmd>") {
		t.Fatalf("stdout=%q", got)
	}
	if !strings.Contains(got, "--dir DIR") {
		t.Fatalf("missing --dir: %q", got)
	}
	for _, env := range []string{"INSTALL_TO_DIR", "INSTALL_GOOS", "INSTALL_GOARCH"} {
		if !strings.Contains(got, env) {
			t.Fatalf("missing %s: %q", env, got)
		}
	}
}

func TestRunInstall_RequiresCmd(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := RunWithWriters(RemoteProfile(), []string{"install"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "install requires <cmd>") {
		t.Fatalf("err=%v", err)
	}
}

func TestRunInstall_NoCandidate(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/empty\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err := RunWithWriters(RemoteProfile(), []string{"install", "missing", "--dir", dir}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), `no install candidate for "missing"`) {
		t.Fatalf("err=%v", err)
	}
	// First stderr line must be the discover marker (no silent scan).
	first := strings.Split(strings.TrimSpace(stderr.String()), "\n")[0]
	if !strings.HasPrefix(first, "[1/4] discover") {
		t.Fatalf("first stderr line=%q want [1/4] discover…\n%s", first, stderr.String())
	}
	if !strings.Contains(stderr.String(), "notice: scanning cmd/ and script/") {
		t.Fatalf("missing scan notice:\n%s", stderr.String())
	}
}

func TestFormatInstallBuildNotice(t *testing.T) {
	got := formatInstallBuildNotice("linux", "amd64", []string{"go", "build", "-o", "/tmp/x", "./cmd/local-agent"}, true)
	want := "GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o /tmp/x ./cmd/local-agent"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveInstallTarget_Flags(t *testing.T) {
	goos, goarch, src, warn := resolveInstallTarget(client.New("http://127.0.0.1:1", ""), "linux", "arm64")
	if goos != "linux" || goarch != "arm64" || src != "flags" || warn != "" {
		t.Fatalf("goos=%s goarch=%s src=%s warn=%q", goos, goarch, src, warn)
	}
}

func TestPlanRemoteInstallDest_SystemPathFallsBack(t *testing.T) {
	dest := planRemoteInstallDest("jq", "/home/u", "/usr/bin/jq", "", "")
	if dest.Primary != "/home/u/.local/bin/jq" {
		t.Fatalf("primary=%s", dest.Primary)
	}
	if dest.SystemWarning == "" || !strings.Contains(dest.SystemWarning, "/usr/bin/jq") {
		t.Fatalf("warning=%q", dest.SystemWarning)
	}
}

func TestPlanRemoteInstallDest_LookPathUnderHome(t *testing.T) {
	dest := planRemoteInstallDest("wrk", "/home/u", "/home/u/go/bin/wrk", "/home/u/go/bin", "/home/u/go")
	if dest.Primary != "/home/u/go/bin/wrk" || !dest.FromPATH {
		t.Fatalf("dest=%+v", dest)
	}
}

func TestFilterExistingExtras(t *testing.T) {
	dest := remoteInstallDest{
		Primary: "/home/u/go/bin/wrk",
		Extras:  []string{"/home/u/.local/bin/wrk", "/home/u/go/bin/wrk"},
	}
	out, err := filterExistingExtras(dest, func(p string) (bool, error) {
		return p == "/home/u/.local/bin/wrk", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Extras) != 1 || out.Extras[0] != "/home/u/.local/bin/wrk" {
		t.Fatalf("extras=%v", out.Extras)
	}
}

func TestCollectStagingBinary_RequiresNamed(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "other"), []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, _, err := collectStagingBinary(dir, "wrk")
	if err == nil || !strings.Contains(err.Error(), "INSTALL_TO_DIR") {
		t.Fatalf("err=%v", err)
	}
}

func TestCollectStagingBinary_NamedAndExtra(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "wrk"), []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "extra"), []byte("y"), 0o755); err != nil {
		t.Fatal(err)
	}
	built, extras, err := collectStagingBinary(dir, "wrk")
	if err != nil {
		t.Fatal(err)
	}
	if built != filepath.Join(dir, "wrk") {
		t.Fatalf("built=%s", built)
	}
	if len(extras) != 1 || extras[0] != "extra" {
		t.Fatalf("extras=%v", extras)
	}
}

func TestRemotePathUnderHome(t *testing.T) {
	if !remotePathUnderHome("/home/u/.local/bin/wrk", "/home/u") {
		t.Fatal("expected under home")
	}
	if remotePathUnderHome("/usr/bin/wrk", "/home/u") {
		t.Fatal("system path")
	}
}

func TestDestWritesLocalBin(t *testing.T) {
	if !destWritesLocalBin(remoteInstallDest{Primary: "/home/u/.local/bin/wrk"}, "/home/u") {
		t.Fatal("primary local bin")
	}
	if destWritesLocalBin(remoteInstallDest{Primary: "/home/u/go/bin/wrk"}, "/home/u") {
		t.Fatal("gopath is not local bin")
	}
}
