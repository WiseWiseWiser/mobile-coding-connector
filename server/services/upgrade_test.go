package services

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// upgradeTestService is one running fixture: a service whose command records
// its own PID so upgrade steps can prove whether it was alive at their moment.
type upgradeTestService struct {
	manager *Manager
	dir     string
	id      string
	pidFile string
	trace   string
}

func newUpgradeTestService(t *testing.T, def ServiceDefinition) *upgradeTestService {
	t.Helper()
	dir := t.TempDir()
	manager := NewManagerAt(dir)
	t.Cleanup(manager.Shutdown)

	fixture := &upgradeTestService{
		manager: manager,
		dir:     dir,
		pidFile: filepath.Join(dir, "service.pid"),
		trace:   filepath.Join(dir, "trace.txt"),
	}
	def.Name = "upgrade-fixture"
	def.WorkingDir = dir
	// The service records its PID and stays alive until stopped.
	def.Command = fmt.Sprintf("echo $$ > %s; sleep 300", fixture.pidFile)
	def.UpgradePreStopCmds = nil
	def.UpgradePostStopCmds = nil

	status, err := manager.CreateOrUpdateNoRestart(def)
	if err != nil {
		t.Fatalf("CreateOrUpdateNoRestart() error = %v", err)
	}
	fixture.id = status.ID
	if _, err := manager.Start(fixture.id); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() { _ = manager.stop(fixture.id, true, true) })
	waitForFile(t, fixture.pidFile)
	return fixture
}

// aliveProbe is a step fragment that records whether the fixture's service
// process was still alive when the step ran.
func (f *upgradeTestService) aliveProbe(label string) string {
	return fmt.Sprintf(
		"if kill -0 \"$(cat %s)\" 2>/dev/null; then echo %s:alive >> %s; else echo %s:dead >> %s; fi",
		f.pidFile, label, f.trace, label, f.trace)
}

func (f *upgradeTestService) appendTrace(label string) string {
	return fmt.Sprintf("echo %s >> %s", label, f.trace)
}

func (f *upgradeTestService) update(t *testing.T, def ServiceDefinition) {
	t.Helper()
	current, ok := f.manager.lookupDefinition(f.id)
	if !ok {
		t.Fatalf("definition %s disappeared", f.id)
	}
	def.ID = f.id
	def.Name = current.Name
	def.Command = current.Command
	def.WorkingDir = current.WorkingDir
	if _, err := f.manager.CreateOrUpdateNoRestart(def); err != nil {
		t.Fatalf("update definition error = %v", err)
	}
}

func (f *upgradeTestService) status(t *testing.T) *ServiceStatus {
	t.Helper()
	status, ok := f.manager.statusByID(f.id)
	if !ok {
		t.Fatalf("status for %s not found", f.id)
	}
	return status
}

func (f *upgradeTestService) traceLines(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(f.trace)
	if err != nil {
		t.Fatalf("read trace: %v", err)
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

func waitForFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", path)
}

// TestUpgradeRunsPreStopLiveAndPostStopStopped pins the ordering contract: the
// pre-stop step must observe a live service and the post-stop step a stopped
// one.
func TestUpgradeRunsPreStopLiveAndPostStopStopped(t *testing.T) {
	fixture := newUpgradeTestService(t, ServiceDefinition{})
	fixture.update(t, ServiceDefinition{
		UpgradePreStopCmds:  []string{fixture.aliveProbe("pre1"), fixture.appendTrace("pre2")},
		UpgradePostStopCmds: []string{fixture.aliveProbe("post1")},
	})

	result, err := fixture.manager.Upgrade(ServiceUpgradeRequest{ID: fixture.id})
	if err != nil {
		t.Fatalf("Upgrade() error = %v", err)
	}
	if result.Service == nil || result.Service.Status != StatusRunning {
		t.Fatalf("service status after upgrade = %#v, want running", result.Service)
	}
	if len(result.Steps) != 3 {
		t.Fatalf("executed steps = %d, want 3", len(result.Steps))
	}
	wantOrder := []string{"pre1:alive", "pre2", "post1:dead"}
	if got := fixture.traceLines(t); !equalStrings(got, wantOrder) {
		t.Fatalf("trace = %v, want %v", got, wantOrder)
	}
	if result.Steps[0].Phase != UpgradePhasePreStop || result.Steps[2].Phase != UpgradePhasePostStop {
		t.Fatalf("step phases = %q/%q, want pre-stop/post-stop", result.Steps[0].Phase, result.Steps[2].Phase)
	}
}

// TestUpgradePreStopFailureLeavesServiceUntouched proves the safety guarantee:
// a broken build must not take the service down or run later phases.
func TestUpgradePreStopFailureLeavesServiceUntouched(t *testing.T) {
	fixture := newUpgradeTestService(t, ServiceDefinition{})
	fixture.update(t, ServiceDefinition{
		UpgradePreStopCmds:  []string{fixture.appendTrace("pre1"), "exit 3", fixture.appendTrace("pre3")},
		UpgradePostStopCmds: []string{fixture.appendTrace("post1")},
	})
	pidBefore := fixture.status(t).PID

	_, err := fixture.manager.Upgrade(ServiceUpgradeRequest{ID: fixture.id})
	if err == nil {
		t.Fatalf("Upgrade() error = nil, want failure")
	}
	var upgradeErr *UpgradeError
	if !errors.As(err, &upgradeErr) {
		t.Fatalf("error = %T, want *UpgradeError: %v", err, err)
	}
	if upgradeErr.Phase != UpgradePhasePreStop || upgradeErr.Index != 2 || upgradeErr.Total != 3 {
		t.Fatalf("error phase/index = %s %d/%d, want pre-stop 2/3", upgradeErr.Phase, upgradeErr.Index, upgradeErr.Total)
	}
	if upgradeErr.ExitCode != 3 {
		t.Fatalf("exit code = %d, want 3", upgradeErr.ExitCode)
	}
	if upgradeErr.Resumed || upgradeErr.ServiceDown {
		t.Fatalf("pre-stop failure reported resume state: %#v", upgradeErr)
	}
	if got := fixture.traceLines(t); !equalStrings(got, []string{"pre1"}) {
		t.Fatalf("trace = %v, want only the first step: later steps must not run", got)
	}
	status := fixture.status(t)
	if status.Status != StatusRunning || status.PID != pidBefore {
		t.Fatalf("service = %s pid %d, want running pid %d (untouched)", status.Status, status.PID, pidBefore)
	}
	if !strings.Contains(err.Error(), "never stopped") {
		t.Fatalf("error message %q should state the service was never stopped", err.Error())
	}
}

// TestUpgradePreStopPipefailAborts guards the `set -eo pipefail` contract: a
// failure anywhere in a pipeline must abort the step.
func TestUpgradePreStopPipefailAborts(t *testing.T) {
	fixture := newUpgradeTestService(t, ServiceDefinition{})
	fixture.update(t, ServiceDefinition{
		UpgradePreStopCmds: []string{"false | true", fixture.appendTrace("after")},
	})

	_, err := fixture.manager.Upgrade(ServiceUpgradeRequest{ID: fixture.id})
	if err == nil {
		t.Fatalf("Upgrade() error = nil, want pipeline failure to abort")
	}
	var upgradeErr *UpgradeError
	if !errors.As(err, &upgradeErr) || upgradeErr.Phase != UpgradePhasePreStop {
		t.Fatalf("error = %v, want pre-stop UpgradeError", err)
	}
	if _, statErr := os.Stat(fixture.trace); statErr == nil {
		t.Fatalf("later step ran despite pipefail failure")
	}
}

// TestUpgradePostStopFailureResumesService covers the outage-safety rule: a
// post-stop failure restarts the service before reporting.
func TestUpgradePostStopFailureResumesService(t *testing.T) {
	fixture := newUpgradeTestService(t, ServiceDefinition{})
	fixture.update(t, ServiceDefinition{
		UpgradePreStopCmds:  []string{fixture.appendTrace("pre1")},
		UpgradePostStopCmds: []string{fixture.aliveProbe("post1"), "exit 7", fixture.appendTrace("post3")},
	})
	pidBefore := fixture.status(t).PID

	_, err := fixture.manager.Upgrade(ServiceUpgradeRequest{ID: fixture.id})
	if err == nil {
		t.Fatalf("Upgrade() error = nil, want failure")
	}
	var upgradeErr *UpgradeError
	if !errors.As(err, &upgradeErr) {
		t.Fatalf("error = %T, want *UpgradeError: %v", err, err)
	}
	if upgradeErr.Phase != UpgradePhasePostStop || upgradeErr.Index != 2 {
		t.Fatalf("error = %s %d/%d, want post-stop 2/3", upgradeErr.Phase, upgradeErr.Index, upgradeErr.Total)
	}
	if upgradeErr.ExitCode != 7 {
		t.Fatalf("exit code = %d, want 7", upgradeErr.ExitCode)
	}
	if !upgradeErr.Resumed || upgradeErr.ServiceDown {
		t.Fatalf("resume state = resumed:%v down:%v, want resumed", upgradeErr.Resumed, upgradeErr.ServiceDown)
	}
	if got := fixture.traceLines(t); !equalStrings(got, []string{"pre1", "post1:dead"}) {
		t.Fatalf("trace = %v, want the failing step to stop the pipeline", got)
	}
	status := fixture.status(t)
	if status.Status != StatusRunning {
		t.Fatalf("service status = %s, want running after resume", status.Status)
	}
	if status.PID == pidBefore {
		t.Fatalf("service pid %d unchanged; want a fresh process after resume", status.PID)
	}
}

// TestUpgradeWithoutStepsOrBinaryIsRejected keeps the "nothing to do" case an
// explicit error instead of a silent restart.
func TestUpgradeWithoutStepsOrBinaryIsRejected(t *testing.T) {
	fixture := newUpgradeTestService(t, ServiceDefinition{})

	_, err := fixture.manager.Upgrade(ServiceUpgradeRequest{ID: fixture.id})
	if err == nil {
		t.Fatalf("Upgrade() error = nil, want rejection")
	}
	if !strings.Contains(err.Error(), "no upgrade steps configured") {
		t.Fatalf("error = %v, want a no-steps message", err)
	}
}

// TestUpgradeStepTimeoutKillsStep covers the bounded-step rule.
func TestUpgradeStepTimeoutKillsStep(t *testing.T) {
	fixture := newUpgradeTestService(t, ServiceDefinition{})
	timeoutSeconds := 1
	fixture.update(t, ServiceDefinition{
		UpgradePreStopCmds:    []string{"sleep 120"},
		UpgradeTimeoutSeconds: &timeoutSeconds,
	})
	pidBefore := fixture.status(t).PID

	started := time.Now()
	_, err := fixture.manager.Upgrade(ServiceUpgradeRequest{ID: fixture.id})
	elapsed := time.Since(started)
	if err == nil {
		t.Fatalf("Upgrade() error = nil, want timeout")
	}
	var upgradeErr *UpgradeError
	if !errors.As(err, &upgradeErr) || !upgradeErr.TimedOut {
		t.Fatalf("error = %v, want a timed-out UpgradeError", err)
	}
	if elapsed > 30*time.Second {
		t.Fatalf("timeout took %s; the step was not killed promptly", elapsed)
	}
	if status := fixture.status(t); status.PID != pidBefore || status.Status != StatusRunning {
		t.Fatalf("service = %s pid %d, want untouched running pid %d", status.Status, status.PID, pidBefore)
	}
}

// TestUpgradeExposesTargetToSteps pins the swap idiom: steps can read the
// resolved target instead of hardcoding it.
func TestUpgradeExposesTargetToSteps(t *testing.T) {
	fixture := newUpgradeTestService(t, ServiceDefinition{})
	target := filepath.Join(fixture.dir, "bin", "app")
	observed := filepath.Join(fixture.dir, "target.txt")
	fixture.update(t, ServiceDefinition{
		UpgradeTarget:      target,
		UpgradePreStopCmds: []string{fmt.Sprintf("printf '%%s' \"$%s\" > %s", EnvUpgradeTarget, observed)},
	})

	if _, err := fixture.manager.Upgrade(ServiceUpgradeRequest{ID: fixture.id}); err != nil {
		t.Fatalf("Upgrade() error = %v", err)
	}
	data, err := os.ReadFile(observed)
	if err != nil {
		t.Fatalf("read observed target: %v", err)
	}
	if got := strings.TrimSpace(string(data)); got != target {
		t.Fatalf("observed %s = %q, want %q", EnvUpgradeTarget, got, target)
	}
}

// TestUpgradeRequestOverridesStoredSteps covers the one-run override flags.
func TestUpgradeRequestOverridesStoredSteps(t *testing.T) {
	fixture := newUpgradeTestService(t, ServiceDefinition{})
	fixture.update(t, ServiceDefinition{
		UpgradePreStopCmds: []string{fixture.appendTrace("stored")},
	})

	if _, err := fixture.manager.Upgrade(ServiceUpgradeRequest{
		ID:          fixture.id,
		PreStopCmds: []string{fixture.appendTrace("override")},
	}); err != nil {
		t.Fatalf("Upgrade() error = %v", err)
	}
	if got := fixture.traceLines(t); !equalStrings(got, []string{"override"}) {
		t.Fatalf("trace = %v, want the request override only", got)
	}
	if def, _ := fixture.manager.lookupDefinition(fixture.id); !equalStrings(def.UpgradePreStopCmds, []string{fixture.appendTrace("stored")}) {
		t.Fatalf("stored steps changed by a one-run override: %v", def.UpgradePreStopCmds)
	}
}

// TestUpgradeStepOutputMirroredToServiceLog keeps failures diagnosable from
// `service logs` after the terminal is gone.
func TestUpgradeStepOutputMirroredToServiceLog(t *testing.T) {
	fixture := newUpgradeTestService(t, ServiceDefinition{})
	fixture.update(t, ServiceDefinition{
		UpgradePreStopCmds: []string{"echo mirrored-line-from-step"},
	})

	if _, err := fixture.manager.Upgrade(ServiceUpgradeRequest{ID: fixture.id}); err != nil {
		t.Fatalf("Upgrade() error = %v", err)
	}
	logPath := fixture.manager.serviceLogPath(fixture.id)
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read service log: %v", err)
	}
	if !strings.Contains(string(data), "mirrored-line-from-step") {
		t.Fatalf("service log does not contain step output:\n%s", data)
	}
}

// TestUpgradeEmitterReportsSectionsAndOutput covers the streaming seam.
func TestUpgradeEmitterReportsSectionsAndOutput(t *testing.T) {
	fixture := newUpgradeTestService(t, ServiceDefinition{})
	fixture.update(t, ServiceDefinition{
		UpgradePreStopCmds: []string{"echo streamed-line"},
	})

	var events []UpgradeEvent
	if _, err := fixture.manager.UpgradeWithEmitter(ServiceUpgradeRequest{ID: fixture.id}, func(event UpgradeEvent) {
		events = append(events, event)
	}); err != nil {
		t.Fatalf("UpgradeWithEmitter() error = %v", err)
	}

	var sections, logs []string
	for _, event := range events {
		switch event.Kind {
		case UpgradeEventSection:
			sections = append(sections, event.Message)
		case UpgradeEventLog:
			logs = append(logs, event.Message)
		}
	}
	if !containsSubstring(sections, "pre-stop 1/1") {
		t.Fatalf("sections = %v, want a pre-stop step section", sections)
	}
	if !containsSubstring(sections, "stopping") || !containsSubstring(sections, "starting") {
		t.Fatalf("sections = %v, want stop and start phases", sections)
	}
	if !containsSubstring(logs, "streamed-line") {
		t.Fatalf("logs = %v, want the step output", logs)
	}
}

// TestUpgradeCmdsNormalizedAndPersisted covers trimming and the empty-clears
// convention shared with ExtraEnv.
func TestUpgradeCmdsNormalizedAndPersisted(t *testing.T) {
	dir := t.TempDir()
	manager := NewManagerAt(dir)
	t.Cleanup(manager.Shutdown)

	timeoutSeconds := 42
	status, err := manager.CreateOrUpdateNoRestart(ServiceDefinition{
		Name:                  "normalize",
		Command:               "sleep 300",
		UpgradePreStopCmds:    []string{"  git fetch  ", "", "   "},
		UpgradePostStopCmds:   []string{" mv /tmp/a /tmp/b "},
		UpgradeTimeoutSeconds: &timeoutSeconds,
	})
	if err != nil {
		t.Fatalf("CreateOrUpdateNoRestart() error = %v", err)
	}
	if !equalStrings(status.UpgradePreStopCmds, []string{"git fetch"}) {
		t.Fatalf("pre-stop cmds = %#v, want trimmed non-empty steps", status.UpgradePreStopCmds)
	}
	if !equalStrings(status.UpgradePostStopCmds, []string{"mv /tmp/a /tmp/b"}) {
		t.Fatalf("post-stop cmds = %#v", status.UpgradePostStopCmds)
	}
	if status.UpgradeTimeoutSeconds == nil || *status.UpgradeTimeoutSeconds != 42 {
		t.Fatalf("timeout = %v, want 42", status.UpgradeTimeoutSeconds)
	}

	// Clearing follows the ExtraEnv convention: an empty list clears.
	cleared, err := manager.CreateOrUpdateNoRestart(ServiceDefinition{
		ID:                  status.ID,
		Name:                "normalize",
		Command:             "sleep 300",
		UpgradePostStopCmds: []string{},
	})
	if err != nil {
		t.Fatalf("clear update error = %v", err)
	}
	if len(cleared.UpgradePreStopCmds) != 0 || len(cleared.UpgradePostStopCmds) != 0 {
		t.Fatalf("cleared cmds = %v / %v, want both empty", cleared.UpgradePreStopCmds, cleared.UpgradePostStopCmds)
	}

	// A negative timeout is a definition error, not a silent clamp.
	negative := -1
	if _, err := manager.CreateOrUpdateNoRestart(ServiceDefinition{
		ID: status.ID, Name: "normalize", Command: "sleep 300", UpgradeTimeoutSeconds: &negative,
	}); err == nil {
		t.Fatalf("negative timeout accepted, want error")
	}
}

// TestUpgradeStepTimeoutResolution covers the default/override/disable matrix.
func TestUpgradeStepTimeoutResolution(t *testing.T) {
	zero := 0
	ten := 10
	cases := []struct {
		name string
		def  ServiceDefinition
		req  ServiceUpgradeRequest
		want time.Duration
	}{
		{name: "default", want: DefaultUpgradeStepTimeout},
		{name: "definition", def: ServiceDefinition{UpgradeTimeoutSeconds: &ten}, want: 10 * time.Second},
		{name: "request wins", def: ServiceDefinition{UpgradeTimeoutSeconds: &ten}, req: ServiceUpgradeRequest{TimeoutSeconds: &zero}, want: 0},
		{name: "disabled", def: ServiceDefinition{UpgradeTimeoutSeconds: &zero}, want: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := upgradeStepTimeout(tc.def, tc.req); got != tc.want {
				t.Fatalf("upgradeStepTimeout() = %s, want %s", got, tc.want)
			}
		})
	}
}

// TestBuildUpgradeShellCommand covers strict mode and PATH restoration.
func TestBuildUpgradeShellCommand(t *testing.T) {
	got := buildUpgradeShellCommand("go build ./...", []string{"PATH=/opt/go/bin:/usr/bin"})
	if !strings.HasPrefix(got, "set -eo pipefail; ") {
		t.Fatalf("command %q must enable strict mode first", got)
	}
	if !strings.Contains(got, "export PATH='/opt/go/bin:/usr/bin'") {
		t.Fatalf("command %q must restore the resolved PATH", got)
	}
	if !strings.HasSuffix(got, "go build ./...") {
		t.Fatalf("command %q must end with the user step", got)
	}
	withoutPath := buildUpgradeShellCommand("true", []string{"HOME=/root"})
	if strings.Contains(withoutPath, "export PATH") {
		t.Fatalf("command %q must not invent a PATH", withoutPath)
	}
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func containsSubstring(values []string, needle string) bool {
	for _, value := range values {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
