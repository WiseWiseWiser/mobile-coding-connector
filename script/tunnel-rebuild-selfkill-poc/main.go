// tunnel-rebuild-selfkill-poc prototypes the remote incident where /ping hangs after
// extension tunnel rebuild during bootstrap (TCP ok, HTTP timeout, server PID unchanged).
//
// Manual test matrix (repo root):
//
//	go run ./script/tunnel-rebuild-selfkill-poc/ run-matrix
//
// Expected:
//   mutex_blocks=true      — rebuild holds utm.mu during slow stop; AddMapping blocks
//   standin_ping_healthy=true — unrelated /ping server stays fast (not mutex starvation)
//   pgrep_skips_server=true   — killOrphanedProcess pattern does not match ai-critic argv
//
// State file: $TMPDIR/ai-critic-tunnel-rebuild-selfkill-poc.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const stateFileName = "ai-critic-tunnel-rebuild-selfkill-poc.json"

type DetectReport struct {
	MutexBlocksDuringStop bool     `json:"mutex_blocks_during_stop"`
	StandInPingHealthy    bool     `json:"standin_ping_healthy"`
	PgrepSkipsServer      bool     `json:"pgrep_skips_server"`
	TestOutput            string   `json:"test_output,omitempty"`
	Notes                 []string `json:"notes,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "reproduce-real":
		runReproduceReal()
	case "detect":
		runDetect(os.Args[2:])
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
	fmt.Fprintf(os.Stderr, `Usage: go run ./script/tunnel-rebuild-selfkill-poc/ <command>

Commands:
  reproduce-real   Run -tags poc tests against unified_tunnel package.
  detect           Print saved detection report JSON.
  run-matrix       reproduce-real + detect; exit 1 unless all signals pass.

Remote log correlation (Jul 8 2026):
  - extension rebuild stop/start around 21:09:50
  - keep-alive /ping healthy at 21:09:46, hangs at 21:09:56 (PID 232724 unchanged)
  - pattern matches State-T freeze, not literal SIGKILL of ai-critic
`)
}

func statePath() string {
	return filepath.Join(os.TempDir(), stateFileName)
}

func repoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %s", wd)
		}
		dir = parent
	}
}

func runReproduceReal() {
	root, err := repoRoot()
	if err != nil {
		fatal(err)
	}
	cmd := exec.Command("go", "test", "-tags", "poc", "-run", "TestPOC", "-count=1", "-v",
		"./server/cloudflare/unified_tunnel/")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	output := string(out)
	fmt.Print(output)

	rep := DetectReport{
		MutexBlocksDuringStop: strings.Contains(output, "TestPOCRebuildBlocksConcurrentAddMapping") && strings.Contains(output, "PASS"),
		StandInPingHealthy:    strings.Contains(output, "TestPOCStandInPingHealthyDuringSlowRebuild") && strings.Contains(output, "PASS"),
		PgrepSkipsServer:      strings.Contains(output, "TestPOCPgrepPatternSkipsStandInServer") && strings.Contains(output, "PASS"),
		TestOutput:            output,
	}
	rep.Notes = []string{
		"Remote PID unchanged + TCP ok + HTTP hang => freeze (often State T), not tunnel SIGKILL of ai-critic",
		"rebuildAndRestartLocked holds utm.mu across stopProcessLocked subprocess work",
		"startProcessLocked: process exited during stop is expected wait-goroutine log interleaving",
	}
	if err != nil {
		rep.Notes = append(rep.Notes, fmt.Sprintf("go test failed: %v", err))
	}
	if err := saveReport(rep); err != nil {
		fatal(err)
	}
	if err != nil {
		os.Exit(1)
	}
}

func saveReport(rep DetectReport) error {
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statePath(), data, 0600)
}

func loadReport() (DetectReport, error) {
	var rep DetectReport
	data, err := os.ReadFile(statePath())
	if err != nil {
		return rep, err
	}
	err = json.Unmarshal(data, &rep)
	return rep, err
}

func runDetect(args []string) {
	fs := flag.NewFlagSet("detect", flag.ExitOnError)
	_ = fs.Parse(args)

	rep, err := loadReport()
	if err != nil {
		fatal(fmt.Errorf("no state at %s — run reproduce-real first: %w", statePath(), err))
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(rep)

	if !rep.MutexBlocksDuringStop || !rep.StandInPingHealthy || !rep.PgrepSkipsServer {
		os.Exit(1)
	}
}

func runMatrix() {
	runReproduceReal()
	rep, err := loadReport()
	if err != nil {
		fatal(err)
	}
	if !rep.MutexBlocksDuringStop || !rep.StandInPingHealthy || !rep.PgrepSkipsServer {
		fatal(fmt.Errorf("matrix failed: mutex=%v ping=%v pgrep=%v",
			rep.MutexBlocksDuringStop, rep.StandInPingHealthy, rep.PgrepSkipsServer))
	}
	fmt.Println("matrix OK: tunnel rebuild blocks tunnel mutex; stand-in /ping unaffected; pgrep skips ai-critic argv")
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}