//go:build darwin

package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// stopIsolatedChrome signals only a process whose native argv confirms the
// exact, unique t.TempDir profile owned by this test. Never touch other Chrome.
func stopIsolatedChrome(root, defaultRoot string) {
	out, err := exec.Command("/bin/ps", "-axo", "pid=,uid=,comm=").Output()
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		pid, e1 := strconv.Atoi(fields[0])
		uid, e2 := strconv.Atoi(fields[1])
		if e1 != nil || e2 != nil || uid != os.Getuid() || !isChromeMainExecutable(strings.Join(fields[2:], " ")) {
			continue
		}
		p, err := inspectChromePID(pid, defaultRoot)
		if err == nil && p.DataDir == root {
			_ = syscall.Kill(pid, syscall.SIGTERM)
		}
	}
}

// Real macOS routing regression: one separate headless Chrome is already
// running, then the production launcher starts a headed Chrome in another
// temporary profile. A unique local URL must not arrive in the headless browser.
func TestChromeRouteDoesNotUseExistingHeadlessProfile(t *testing.T) {
	if os.Getenv("AI_CRITIC_TEST_PROFILE_ROUTE") != "1" {
		t.Skip("isolated headed Chrome routing test disabled")
	}
	headlessRoot, _ := filepath.EvalSymlinks(t.TempDir())
	targetRoot, _ := filepath.EvalSymlinks(t.TempDir())
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	defaultRoot, err := filepath.EvalSymlinks(filepath.Join(home, "Library/Application Support/Google/Chrome"))
	if err != nil {
		t.Fatal(err)
	}
	pkg := makeAgentPackage(t, targetRoot)
	writeProfile(t, targetRoot, "Default", fmt.Sprintf(`{"a":{"path":%q}}`, pkg))
	// A fresh temporary directory is not the user's existing initialized profile.
	if err := os.WriteFile(filepath.Join(targetRoot, "First Run"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BROWSER_AGENT_CHROME_USER_DATA_DIR", targetRoot)
	t.Setenv("BROWSER_AGENT_CHROME_PROFILE", "Default")

	headless := exec.Command(chromeBundledBinary, "--headless=new", "--no-first-run", "--user-data-dir="+headlessRoot, "--remote-debugging-port=0", "about:blank")
	if err := headless.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = headless.Process.Kill(); _ = headless.Wait() }()
	defer stopIsolatedChrome(targetRoot, defaultRoot)
	deadline := time.Now().Add(20 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(headlessRoot, "DevToolsActivePort")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("headless test Chrome did not start")
		}
		time.Sleep(100 * time.Millisecond)
	}
	bytes, err := os.ReadFile(filepath.Join(headlessRoot, "DevToolsActivePort"))
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(strings.Split(string(bytes), "\n")[0])
	if err != nil {
		t.Fatal(err)
	}
	requests := make(chan string, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/isolated-startup-test" {
			select {
			case requests <- r.URL.RawQuery:
			default:
			}
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<title>isolated startup test</title>"))
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	url := server.URL + "/isolated-startup-test"
	logPath := filepath.Join(t.TempDir(), "chrome-test.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatal(err)
	}
	defer logFile.Close()
	var target *exec.Cmd
	defer func() {
		if target != nil && target.Process != nil {
			_ = target.Process.Kill()
			_ = target.Wait()
		}
	}()
	launch := func(binary string, args []string) error {
		args = append([]string{"--remote-debugging-port=0"}, args...)
		target = exec.Command(binary, args...)
		target.Stdout, target.Stderr = logFile, logFile
		return target.Start()
	}
	if err := openChromeProfileWith(ctx, url, false, launch); err != nil {
		t.Fatal(err)
	}
	select {
	case <-requests:
	case <-ctx.Done():
		logs, _ := os.ReadFile(logPath)
		t.Fatalf("target Chrome did not open the local test page: %v; test Chrome log: %s", ctx.Err(), logs)
	}
	lock, err := os.Readlink(filepath.Join(targetRoot, "SingletonLock"))
	if err != nil {
		t.Fatalf("target profile lock: %v", err)
	}
	cut := strings.LastIndexByte(lock, '-')
	if cut < 0 {
		t.Fatal("invalid target profile lock")
	}
	pid, err := strconv.Atoi(lock[cut+1:])
	if err != nil {
		t.Fatal(err)
	}
	owner, err := inspectChromePID(pid, defaultRoot)
	if err != nil || owner.Headless || owner.DataDir != targetRoot {
		t.Fatalf("target owner=%+v err=%v", owner, err)
	}
	// A second request for the same user-data root must reuse that browser,
	// even while the unrelated headless browser remains running.
	second := exec.Command(chromeBundledBinary, "--user-data-dir="+targetRoot, "--profile-directory=Default", url+"?reuse=1")
	second.Stdout, second.Stderr = logFile, logFile
	if err := second.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = second.Process.Kill(); _ = second.Wait() }()
	received := false
	for !received {
		select {
		case query := <-requests:
			received = query == "reuse=1"
		case <-ctx.Done():
			t.Fatal("running profile did not receive the second request")
		}
	}
	again, err := os.Readlink(filepath.Join(targetRoot, "SingletonLock"))
	if err != nil || again != lock {
		t.Fatalf("profile owner changed: %s -> %s; %v", lock, again, err)
	}
	response, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/json/list", port))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var targets []struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(response.Body).Decode(&targets); err != nil {
		t.Fatal(err)
	}
	for _, target := range targets {
		if strings.Contains(target.URL, "/isolated-startup-test") {
			t.Fatal("launch request was delivered to unrelated headless Chrome")
		}
	}
	t.Log("unique URL opened in isolated headed profile; unrelated headless profile was not used")
}
