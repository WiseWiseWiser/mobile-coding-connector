// keepalive-tty-stop-poc prototypes the remote hang: keep-alive + server on a
// shared controlling terminal, server child frozen in State T while TCP :port
// still accepts connections and GET /ping hangs.
//
// Manual test matrix (run from repo root):
//
//	go run ./script/keepalive-tty-stop-poc/ stop
//	go run ./script/keepalive-tty-stop-poc/ reproduce --mode attached
//	go run ./script/keepalive-tty-stop-poc/ detect
//	go run ./script/keepalive-tty-stop-poc/ stop
//
//	go run ./script/keepalive-tty-stop-poc/ reproduce --mode detached
//	go run ./script/keepalive-tty-stop-poc/ detect
//	go run ./script/keepalive-tty-stop-poc/ stop
//
//	go run ./script/keepalive-tty-stop-poc/ reproduce-keepalive --mode attached
//	go run ./script/keepalive-tty-stop-poc/ detect
//	go run ./script/keepalive-tty-stop-poc/ stop
//
// Expected:
//   attached → detect reports hung_signature=true (State T + TCP ok + ping fail)
//   detached → detect reports hung_signature=false after trigger attempt
//
// State file: $TMPDIR/ai-critic-keepalive-tty-stop-poc.json
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/creack/pty"
)

const stateFileName = "ai-critic-keepalive-tty-stop-poc.json"

type RunState struct {
	Mode       string `json:"mode"`
	Port       int    `json:"port"`
	ParentPID  int    `json:"parent_pid"`
	ServerPID  int    `json:"server_pid"`
	PTY        string `json:"pty,omitempty"`
	Binary     string `json:"binary,omitempty"`
	StartedAt  string `json:"started_at"`
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "server-child":
		runServerChild()
	case "detect":
		runDetect(os.Args[2:])
	case "reproduce":
		runReproduce(os.Args[2:])
	case "reproduce-keepalive":
		runReproduceKeepalive(os.Args[2:])
	case "stop":
		runStop()
	case "run-matrix":
		runMatrix()
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: go run ./script/keepalive-tty-stop-poc/ <command>

Commands:
  reproduce [--mode attached|detached]
      Mini keep-alive parent on a pty; spawns server-child with old (attached)
      or fixed (detached) proc attrs; triggers job-control stop on attached path.
  reproduce-keepalive [--mode attached|detached] [--binary PATH]
      Same hang attempt using the real ai-critic keep-alive binary when built.
  detect [--port N] [--pid N]
      Report process state, TCP probe, /ping probe, hung_signature separately.
  stop
      Tear down processes recorded in the POC state file.
  run-matrix
      Automated local proof: attached (expect hung) then detached (expect healthy).
  server-child --port P --mode attached|detached
      Internal: minimal GET /ping -> pong server used by reproduce.
`)
}

func statePath() string {
	return filepath.Join(os.TempDir(), stateFileName)
}

func loadState() (*RunState, error) {
	data, err := os.ReadFile(statePath())
	if err != nil {
		return nil, err
	}
	var st RunState
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

func saveState(st *RunState) error {
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statePath(), data, 0600)
}

func runServerChild() {
	fs := flag.NewFlagSet("server-child", flag.ExitOnError)
	port := fs.Int("port", 0, "listen port")
	mode := fs.String("mode", "attached", "attached|detached")
	_ = fs.Parse(os.Args[2:])
	if *port <= 0 {
		fmt.Fprintln(os.Stderr, "server-child: --port required")
		os.Exit(2)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "pong")
	})
	srv := &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", *port), Handler: mux}
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "server-child listen: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "server-child ready pid=%d port=%d mode=%s\n", os.Getpid(), *port, *mode)
	go func() {
		// Background goroutine writes to stdout after ready — on attached mode
		// with a controlling pty this can draw SIGTTOU and freeze the process.
		time.Sleep(300 * time.Millisecond)
		fmt.Fprintf(os.Stdout, "[server-child] periodic log pid=%d mode=%s\n", os.Getpid(), *mode)
	}()
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "server-child serve: %v\n", err)
		os.Exit(1)
	}
}

func childProcAttr(mode string) *syscall.SysProcAttr {
	if mode == "detached" {
		return &syscall.SysProcAttr{Setsid: true, Setpgid: true}
	}
	return &syscall.SysProcAttr{Setpgid: true}
}

func pickPort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

func selfExecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

func waitPing(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	url := fmt.Sprintf("http://127.0.0.1:%d/ping", port)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK && strings.TrimSpace(string(body)) == "pong" {
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("ping not ready on port %d within %v", port, timeout)
}

type miniScenario struct {
	state *RunState
	ptmx  *os.File
	cmd   *exec.Cmd
}

func (s *miniScenario) cleanup() {
	if s.cmd != nil && s.cmd.Process != nil {
		killTree(s.cmd.Process.Pid)
	}
	if s.ptmx != nil {
		_ = s.ptmx.Close()
	}
}

func setupMiniReproduce(mode string) (*miniScenario, error) {
	if mode != "attached" && mode != "detached" {
		return nil, fmt.Errorf("mode must be attached or detached")
	}
	port, err := pickPort()
	if err != nil {
		return nil, err
	}
	exe, err := selfExecutable()
	if err != nil {
		return nil, err
	}

	var ptmx *os.File
	cmd := exec.Command(exe, "server-child", "--port", strconv.Itoa(port), "--mode", mode)
	switch mode {
	case "attached":
		ptmx, err = pty.Start(exec.Command("sleep", "3600"))
		if err != nil {
			return nil, err
		}
		cmd.SysProcAttr = childProcAttr(mode)
		cmd.Stdin = ptmx
		cmd.Stdout = ptmx
		cmd.Stderr = ptmx
	default:
		// No controlling terminal — models setsid/nohup-launched keep-alive.
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.Stdin = nil
		devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
		if err != nil {
			return nil, err
		}
		cmd.Stdout = devNull
		cmd.Stderr = devNull
	}
	if err := cmd.Start(); err != nil {
		if ptmx != nil {
			ptmx.Close()
		}
		return nil, err
	}
	if err := waitPing(port, 8*time.Second); err != nil {
		killTree(cmd.Process.Pid)
		ptmx.Close()
		return nil, err
	}
	if mode == "attached" {
		pgid, err := syscall.Getpgid(cmd.Process.Pid)
		if err != nil {
			s := &miniScenario{ptmx: ptmx, cmd: cmd}
			s.cleanup()
			return nil, err
		}
		if err := syscall.Kill(-pgid, syscall.SIGTSTP); err != nil {
			s := &miniScenario{ptmx: ptmx, cmd: cmd}
			s.cleanup()
			return nil, err
		}
		fmt.Printf("triggered SIGTSTP on server pgid=%d (attached reproduction)\n", pgid)
	} else {
		time.Sleep(800 * time.Millisecond)
		fmt.Println("detached mode: no SIGTSTP sent; server should keep serving /ping")
	}
	ptyName := ""
	if ptmx != nil {
		ptyName = ptmx.Name()
	}
	st := &RunState{
		Mode:      mode,
		Port:      port,
		ParentPID: os.Getpid(),
		ServerPID: cmd.Process.Pid,
		PTY:       ptyName,
		StartedAt: time.Now().Format(time.RFC3339),
	}
	if err := saveState(st); err != nil {
		s := &miniScenario{state: st, ptmx: ptmx, cmd: cmd}
		s.cleanup()
		return nil, err
	}
	return &miniScenario{state: st, ptmx: ptmx, cmd: cmd}, nil
}

func runReproduce(args []string) {
	fs := flag.NewFlagSet("reproduce", flag.ExitOnError)
	mode := fs.String("mode", "attached", "attached|detached")
	hold := fs.Duration("hold", 20*time.Second, "keep POC alive before exit (0 for immediate return)")
	_ = fs.Parse(args)

	sc, err := setupMiniReproduce(*mode)
	if err != nil {
		fatal(err)
	}
	defer sc.cleanup()

	fmt.Printf("reproduce mode=%s port=%d server_pid=%d state=%s\n",
		sc.state.Mode, sc.state.Port, sc.state.ServerPID, statePath())
	if *hold > 0 {
		fmt.Printf("holding POC alive for %v — run detect from another terminal or use run-matrix\n", *hold)
		time.Sleep(*hold)
	}
}

func findAICriticBinary(explicit string) (string, error) {
	if explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			return "", err
		}
		return explicit, nil
	}
	candidates := []string{
		"./ai-critic-server-linux-amd64",
		"./ai-critic-server",
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c, nil
		}
	}
	return "", fmt.Errorf("no ai-critic binary found; build first or pass --binary")
}

func runReproduceKeepalive(args []string) {
	fs := flag.NewFlagSet("reproduce-keepalive", flag.ExitOnError)
	mode := fs.String("mode", "attached", "attached|detached")
	binary := fs.String("binary", "", "path to ai-critic binary")
	serverPort := fs.Int("server-port", 0, "managed server port (0 = auto)")
	_ = fs.Parse(args)
	if *mode != "attached" && *mode != "detached" {
		fmt.Fprintln(os.Stderr, "--mode must be attached or detached")
		os.Exit(2)
	}

	bin, err := findAICriticBinary(*binary)
	if err != nil {
		fatal(err)
	}

	port := *serverPort
	if port == 0 {
		port, err = pickPort()
		if err != nil {
			fatal(err)
		}
	}

	ptmx, err := pty.Start(exec.Command("sleep", "3600"))
	if err != nil {
		fatal(err)
	}
	defer ptmx.Close()

	argsKeep := []string{"keep-alive", "--port", strconv.Itoa(port), "--startup-timeout", "30s", "--forever", "--log", "no"}
	cmd := exec.Command(bin, argsKeep...)
	if *mode == "detached" {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	}
	cmd.Stdin = ptmx
	cmd.Stdout = ptmx
	cmd.Stderr = ptmx
	if err := cmd.Start(); err != nil {
		fatal(err)
	}

	if err := waitPing(port, 45*time.Second); err != nil {
		fatal(err)
	}

	// Find managed server child PID via port listener.
	serverPID, err := pidListeningOn(port)
	if err != nil {
		fatal(err)
	}

	if *mode == "attached" {
		pgid, err := syscall.Getpgid(serverPID)
		if err != nil {
			fatal(err)
		}
		if err := syscall.Kill(-pgid, syscall.SIGTSTP); err != nil {
			fatal(err)
		}
		fmt.Printf("triggered SIGTSTP on real server pgid=%d\n", pgid)
	} else {
		time.Sleep(800 * time.Millisecond)
		fmt.Println("detached keep-alive: no SIGTSTP sent")
	}

	st := &RunState{
		Mode:      *mode,
		Port:      port,
		ParentPID: cmd.Process.Pid,
		ServerPID: serverPID,
		PTY:       ptmx.Name(),
		Binary:    bin,
		StartedAt: time.Now().Format(time.RFC3339),
	}
	if err := saveState(st); err != nil {
		fatal(err)
	}
	fmt.Printf("reproduce-keepalive mode=%s port=%d keepalive_pid=%d server_pid=%d\n", *mode, port, cmd.Process.Pid, serverPID)
}

func pidListeningOn(port int) (int, error) {
	out, err := exec.Command("lsof", "-nP", "-iTCP:"+strconv.Itoa(port), "-sTCP:LISTEN", "-t").Output()
	if err != nil {
		return 0, fmt.Errorf("lsof port %d: %w", port, err)
	}
	line := strings.TrimSpace(string(out))
	if line == "" {
		return 0, fmt.Errorf("no listener on port %d", port)
	}
	first := strings.Split(line, "\n")[0]
	return strconv.Atoi(strings.TrimSpace(first))
}

type DetectReport struct {
	Port           int    `json:"port"`
	PID            int    `json:"pid"`
	ProcessState   string `json:"process_state"`
	ProcessStopped bool   `json:"process_stopped"`
	TCPOK          bool   `json:"tcp_ok"`
	PingOK         bool   `json:"ping_ok"`
	PingLatencyMS  int64  `json:"ping_latency_ms,omitempty"`
	HungSignature  bool   `json:"hung_signature"`
	Notes          string `json:"notes,omitempty"`
}

func buildDetectReport(port, pid int) DetectReport {
	report := DetectReport{Port: port, PID: pid}
	if pid > 0 {
		report.ProcessState = processState(pid)
		report.ProcessStopped = isStoppedState(report.ProcessState)
	}
	if port > 0 {
		report.TCPOK = tcpOK(port)
		ok, latency := pingOK(port)
		report.PingOK = ok
		report.PingLatencyMS = latency.Milliseconds()
	}
	report.HungSignature = report.TCPOK && !report.PingOK && report.ProcessStopped
	if report.HungSignature {
		report.Notes = "matches remote incident: TCP accepts, /ping hangs, process State T"
	} else if report.TCPOK && !report.PingOK {
		report.Notes = "ping hung but process not in stopped state (different failure mode)"
	}
	return report
}

func runDetect(args []string) {
	fs := flag.NewFlagSet("detect", flag.ExitOnError)
	portFlag := fs.Int("port", 0, "server port")
	pidFlag := fs.Int("pid", 0, "server pid")
	_ = fs.Parse(args)

	port := *portFlag
	pid := *pidFlag
	if st, err := loadState(); err == nil {
		if port == 0 {
			port = st.Port
		}
		if pid == 0 {
			pid = st.ServerPID
		}
	}

	report := buildDetectReport(port, pid)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(report)
	if report.HungSignature {
		os.Exit(1)
	}
}

func processState(pid int) string {
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "state=").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func isStoppedState(state string) bool {
	return strings.ContainsAny(state, "T")
}

func tcpOK(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func pingOK(port int) (bool, time.Duration) {
	start := time.Now()
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/ping", port))
	if err != nil {
		return false, time.Since(start)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, time.Since(start)
	}
	return resp.StatusCode == http.StatusOK && strings.TrimSpace(string(body)) == "pong", time.Since(start)
}

func runStop() {
	st, err := loadState()
	if err != nil {
		fmt.Println("no state file; nothing to stop")
		return
	}
	killTree(st.ServerPID)
	if st.ParentPID > 0 && st.ParentPID != os.Getpid() {
		killTree(st.ParentPID)
	}
	_ = os.Remove(statePath())
	fmt.Println("stopped POC processes")
}

func killTree(pid int) {
	if pid <= 0 {
		return
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
	pgid, err := syscall.Getpgid(pid)
	if err == nil {
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	}
}

func runMatrix() {
	runStop()

	fmt.Println("=== matrix: attached (expect hung_signature) ===")
	attached, err := setupMiniReproduce("attached")
	if err != nil {
		fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	attachedReport := buildDetectReport(attached.state.Port, attached.state.ServerPID)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(attachedReport)
	attached.cleanup()
	fmt.Printf("attached hung_signature=%v\n", attachedReport.HungSignature)

	fmt.Println("=== matrix: detached (expect healthy) ===")
	detached, err := setupMiniReproduce("detached")
	if err != nil {
		fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	detachedReport := buildDetectReport(detached.state.Port, detached.state.ServerPID)
	_ = enc.Encode(detachedReport)
	detached.cleanup()
	runStop()
	fmt.Printf("detached hung_signature=%v ping_ok=%v\n", detachedReport.HungSignature, detachedReport.PingOK)

	if !attachedReport.HungSignature || detachedReport.HungSignature || !detachedReport.PingOK {
		fatal(fmt.Errorf("matrix failed: attached_hung=%v detached_healthy=%v",
			attachedReport.HungSignature, detachedReport.PingOK && !detachedReport.HungSignature))
	}
	fmt.Println("matrix passed")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

// Linux-only helper kept for future /proc parsing; macOS uses ps in processState.
func procStatusState(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return ""
	}
	for _, line := range bytes.Split(data, []byte("\n")) {
		if bytes.HasPrefix(line, []byte("State:")) {
			fields := bytes.Fields(line)
			if len(fields) >= 2 {
				return string(fields[1])
			}
		}
	}
	return ""
}