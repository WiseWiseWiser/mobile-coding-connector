package agentcli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
