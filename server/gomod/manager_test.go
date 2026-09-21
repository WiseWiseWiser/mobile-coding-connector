package gomod

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStartServesProxy(t *testing.T) {
	tmp := t.TempDir()
	setTestHome(t, tmp)
	root := newTestRoot(t)

	m, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetRoot(root); err != nil {
		t.Fatal(err)
	}
	if err := m.SetPort(testPort()); err != nil {
		t.Fatal(err)
	}
	kv, err := m.Start()
	if err != nil {
		t.Fatal(err)
	}
	if kv["status"] != "running" {
		t.Fatalf("status: %v", kv)
	}
	defer m.Stop()

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/github.com/!burnt!sushi/toml/@v/v1.6.0.zip", kv["port"]))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || string(body) != "ZIPBYTES-v1.6.0" {
		t.Fatalf("http: %d %q", resp.StatusCode, string(body))
	}
}

func TestStartTwiceIdempotent(t *testing.T) {
	tmp := t.TempDir()
	setTestHome(t, tmp)
	root := newTestRoot(t)
	m, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}
	_ = m.SetRoot(root)
	_ = m.SetPort(testPort())
	if _, err := m.Start(); err != nil {
		t.Fatal(err)
	}
	defer m.Stop()
	kv, err := m.Start()
	if err != nil {
		t.Fatalf("second start should be idempotent: %v", err)
	}
	if kv["status"] != "running" {
		t.Fatalf("status: %v", kv)
	}
}

func TestStopWhenNotRunningWarns(t *testing.T) {
	tmp := t.TempDir()
	setTestHome(t, tmp)
	m, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}
	kv, err := m.Stop()
	if err != nil {
		t.Fatal(err)
	}
	if kv != nil {
		t.Fatalf("stop when stopped should return nil kv (caller warns), got %v", kv)
	}
}

func TestStartFailsWithoutRoot(t *testing.T) {
	tmp := t.TempDir()
	setTestHome(t, tmp)
	m, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Start(); err == nil {
		t.Fatal("start with missing root should fail")
	} else if !contains(err.Error(), "does not exist or has no modules") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnableDisableRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	setTestHome(t, tmp)
	m, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Enable(false); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(tmp, ".ai-critic", "go", ConfigFileName))
	var cfg Config
	_ = json.Unmarshal(data, &cfg)
	if !cfg.Enabled {
		t.Fatal("enable should persist enabled=true")
	}
	if _, err := m.Disable(); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(filepath.Join(tmp, ".ai-critic", "go", ConfigFileName))
	_ = json.Unmarshal(data, &cfg)
	if cfg.Enabled {
		t.Fatal("disable should persist enabled=false")
	}
}

func TestBootAutoStart(t *testing.T) {
	tmp := t.TempDir()
	setTestHome(t, tmp)
	root := newTestRoot(t)

	// Persist enabled=true as if the user ran enable earlier.
	m0, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}
	if err := m0.SetRoot(root); err != nil {
		t.Fatal(err)
	}
	if _, err := m0.Enable(false); err != nil {
		t.Fatal(err)
	}

	// Simulate server reboot: a fresh manager reads the flag and binds.
	m, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}
	m.BootAutoStart()
	defer m.Stop()
	kv := m.Status()
	if kv["status"] != "running" {
		t.Fatalf("boot auto-start: status=%v", kv)
	}
	if kv["auto_start"] != "enabled" {
		t.Fatalf("auto_start: %v", kv)
	}
}

func TestTailLog(t *testing.T) {
	tmp := t.TempDir()
	setTestHome(t, tmp)
	m, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}
	root := newTestRoot(t)
	_ = m.SetRoot(root)
	_ = m.SetPort(testPort())
	if _, err := m.Start(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	m.Stop()
	out, err := m.TailLog(10)
	if err != nil {
		t.Fatal(err)
	}
	if out == "" {
		t.Fatal("log should contain startup lines")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// testPort returns a port in a high range unlikely to collide.
func testPort() int {
	return 29870 + (os.Getpid() % 100)
}

// setTestHome points os.UserHomeDir at tmp via HOME env (darwin/linux).
func setTestHome(t *testing.T, tmp string) {
	t.Helper()
	t.Setenv("HOME", tmp)
}
