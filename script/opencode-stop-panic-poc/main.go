// opencode-stop-panic-poc reproduces the remote ai-critic crash:
//
//	panic: close of closed channel
//	exposed_opencode.Stop() via health_check restart (port 4096 unreachable)
//
// Manual test matrix (repo root):
//
//	go run ./script/opencode-stop-panic-poc/ run-matrix
//
//	go run ./script/opencode-stop-panic-poc/ reproduce-model
//	go run ./script/opencode-stop-panic-poc/ reproduce-real
//	go run ./script/opencode-stop-panic-poc/ detect
//
// Expected (after fix):
//   reproduce-model → panic_signature=true (unsafe Stop replica — bug class demo)
//   reproduce-real  → TestPOC* pass (real Stop is idempotent; no panic on closed StopChan)
//
// State file: $TMPDIR/ai-critic-opencode-stop-panic-poc.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

const stateFileName = "ai-critic-opencode-stop-panic-poc.json"

type DetectReport struct {
	ModelPanicSignature bool     `json:"model_panic_signature"`
	RealTestsPassed     bool     `json:"real_tests_passed"`
	RealTestOutput      string   `json:"real_test_output,omitempty"`
	Notes               []string `json:"notes,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "reproduce-model":
		runReproduceModel()
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
	fmt.Fprintf(os.Stderr, `Usage: go run ./script/opencode-stop-panic-poc/ <command>

Commands:
  reproduce-model
      Mini replica of exposed_opencode.Stop(): close StopChan without checking closed.
  reproduce-real
      Run -tags poc tests against the real exposed_opencode package.
  detect
      Summarize model + real reproduction results.
  run-matrix
      reproduce-model then reproduce-real; exit 1 unless signatures match expectation.
`)
}

func statePath() string {
	return filepath.Join(os.TempDir(), stateFileName)
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

// buggyStop mirrors exposed_opencode.Stop channel handling (no closed-channel guard).
type buggyManager struct {
	stopChan chan struct{}
}

var (
	buggy     *buggyManager
	buggyMu   sync.Mutex
)

func buggyStop() {
	buggyMu.Lock()
	defer buggyMu.Unlock()
	if buggy != nil && buggy.stopChan != nil {
		close(buggy.stopChan)
		buggy.stopChan = nil
	}
}

func runReproduceModel() {
	buggyMu.Lock()
	ch := make(chan struct{})
	close(ch)
	buggy = &buggyManager{stopChan: ch}
	buggyMu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("model panic_signature=true panic=%v\n", r)
			_ = saveReport(DetectReport{
				ModelPanicSignature: true,
				Notes: []string{
					"Stale StopChan: non-nil pointer to already-closed channel",
					"Matches remote: panic: close of closed channel in exposed_opencode.Stop",
				},
			})
			return
		}
		fmt.Println("model panic_signature=false (unexpected)")
		os.Exit(1)
	}()

	buggyStop()
	fmt.Println("model panic_signature=false (unexpected)")
	os.Exit(1)
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
		"./server/agents/opencode/exposed_opencode/")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	output := string(out)
	fmt.Print(output)

	rep, _ := loadReport()
	rep.RealTestOutput = output
	rep.RealTestsPassed = err == nil
	if rep.RealTestsPassed {
		rep.Notes = append(rep.Notes, "Real package POC tests passed (stale closed channel panics; concurrent Stop safe)")
	} else {
		rep.Notes = append(rep.Notes, "Real package POC tests failed")
	}
	if err := saveReport(rep); err != nil {
		fatal(err)
	}
	if !rep.RealTestsPassed {
		os.Exit(1)
	}
}

func runDetect(args []string) {
	fs := flag.NewFlagSet("detect", flag.ExitOnError)
	_ = fs.Parse(args)

	rep, err := loadReport()
	if err != nil {
		fatal(fmt.Errorf("no state file %s — run reproduce-model / reproduce-real first: %w", statePath(), err))
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(rep)

	ok := rep.ModelPanicSignature && rep.RealTestsPassed
	if !ok {
		os.Exit(1)
	}
}

func runMatrix() {
	runReproduceModel()
	runReproduceReal()
	rep, err := loadReport()
	if err != nil {
		fatal(err)
	}
	if !rep.ModelPanicSignature || !rep.RealTestsPassed {
		fatal(fmt.Errorf("matrix failed: model_panic=%v real_tests=%v", rep.ModelPanicSignature, rep.RealTestsPassed))
	}
	fmt.Println("matrix OK: model shows bug class; real exposed_opencode.Stop is idempotent")
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}