package services

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// UpgradeEventKind is the shape of one streamed upgrade event.
type UpgradeEventKind string

const (
	UpgradeEventSection UpgradeEventKind = "section"
	UpgradeEventLog     UpgradeEventKind = "log"
)

// UpgradeEvent is one incremental upgrade event. A nil emitter means the
// caller wants no incremental reporting.
type UpgradeEvent struct {
	Kind    UpgradeEventKind
	Message string
}

// UpgradeEmitter receives incremental upgrade events. Implementations must be
// safe for concurrent use: step stdout and stderr are forwarded from separate
// goroutines.
type UpgradeEmitter func(UpgradeEvent)

// EnvUpgradeTarget exposes the resolved upgrade target path to every step, so
// a swap step can move a freshly built artifact into place without repeating
// the path.
const EnvUpgradeTarget = "REMOTE_AGENT_UPGRADE_TARGET"

// upgradeTailLines bounds the captured step output kept for error reporting.
const upgradeTailLines = 20

// Upgrade runs the configured upgrade pipeline for a service.
func (m *Manager) Upgrade(req ServiceUpgradeRequest) (*ServiceUpgradeResult, error) {
	return m.upgrade(req, nil)
}

// UpgradeWithEmitter runs the upgrade pipeline and reports each phase and
// every step output line as it happens.
func (m *Manager) UpgradeWithEmitter(req ServiceUpgradeRequest, emit UpgradeEmitter) (*ServiceUpgradeResult, error) {
	return m.upgrade(req, emit)
}

// upgrade runs: pre-stop steps (service live) -> stop -> artifact move ->
// post-stop steps (service down) -> start.
//
// A pre-stop failure aborts with the service untouched. A failure once the
// service is down aborts but restarts the service first, so a broken upgrade
// does not become an outage; only a failed restart leaves it stopped.
func (m *Manager) upgrade(req ServiceUpgradeRequest, emit UpgradeEmitter) (*ServiceUpgradeResult, error) {
	id := strings.TrimSpace(req.ID)
	if id == "" {
		return nil, fmt.Errorf("service id is required")
	}
	if err := m.rejectSystemServiceEdit(id, "upgraded"); err != nil {
		return nil, err
	}

	def, ok := m.lookupDefinition(id)
	if !ok {
		return nil, fmt.Errorf("service not found")
	}

	preCmds := normalizeUpgradeCmds(def.UpgradePreStopCmds)
	if req.PreStopCmds != nil {
		preCmds = normalizeUpgradeCmds(req.PreStopCmds)
	}
	postCmds := normalizeUpgradeCmds(def.UpgradePostStopCmds)
	if req.PostStopCmds != nil {
		postCmds = normalizeUpgradeCmds(req.PostStopCmds)
	}
	timeout := upgradeStepTimeout(def, req)

	tmpPath := strings.TrimSpace(req.TmpPath)
	localBase := normalizeUpgradeLocalBase(req.LocalBase)
	hasBinary := tmpPath != "" || localBase != ""

	if !hasBinary && len(preCmds) == 0 && len(postCmds) == 0 {
		return nil, fmt.Errorf("no upgrade steps configured: pass a binary, or set --upgrade-pre-stop-cmd/--upgrade-post-stop-cmd")
	}
	if hasBinary {
		if tmpPath == "" {
			return nil, fmt.Errorf("temporary upload path is required")
		}
		if localBase == "" {
			return nil, fmt.Errorf("local binary basename is required")
		}
	}

	targetPath := def.UpgradeTarget
	remembered := def.UpgradeTarget
	if hasBinary {
		target, err := m.selectServiceUpgradeTarget(id, localBase, req.Target)
		if err != nil {
			return nil, err
		}
		targetPath = target.Path
		remembered = target.Remembered
	}

	emitSection(emit, upgradeHeader(def, preCmds, postCmds, targetPath))

	// The resolved target is advertised to every step so a build/swap pair can
	// share one path. It is absent when no target is known.
	stepEnv := map[string]string{}
	if strings.TrimSpace(targetPath) != "" {
		stepEnv[EnvUpgradeTarget] = targetPath
	}

	runner := &upgradeRunner{manager: m, def: def, timeout: timeout, extraEnv: stepEnv, emit: emit}
	steps := make([]ServiceUpgradeStep, 0, len(preCmds)+len(postCmds))

	// Phase 1: pre-stop steps run against the live service.
	for i, command := range preCmds {
		step := ServiceUpgradeStep{Phase: UpgradePhasePreStop, Index: i + 1, Total: len(preCmds), Command: command}
		emitSection(emit, upgradeStepTitle(step))
		if err := runner.run(step); err != nil {
			step.ExitCode = upgradeExitCode(err)
			return nil, err
		}
		steps = append(steps, step)
	}

	// Phase 2: stop. Post-stop steps assume a stopped service, so a stop
	// failure aborts before they run.
	emitSection(emit, fmt.Sprintf("stopping %s", def.Name))
	if err := m.stop(id, true, true); err != nil {
		return nil, &UpgradeError{Phase: "stop", ExitCode: -1, Cause: err}
	}

	// Phase 3: move the uploaded artifact while the service is down, so the
	// managed binary is in place before any post-stop step inspects it.
	if hasBinary {
		emitSection(emit, fmt.Sprintf("installing %s", localBase))
		if err := moveServiceUpgradeFile(tmpPath, targetPath); err != nil {
			return nil, m.resumeAfterUpgradeFailure(id, &UpgradeError{
				Phase:    "install",
				ExitCode: -1,
				Cause:    fmt.Errorf("move %s to %s: %w", tmpPath, targetPath, err),
			})
		}
	}

	// Phase 4: post-stop steps run only while the service is stopped.
	for i, command := range postCmds {
		step := ServiceUpgradeStep{Phase: UpgradePhasePostStop, Index: i + 1, Total: len(postCmds), Command: command}
		emitSection(emit, upgradeStepTitle(step))
		if err := runner.run(step); err != nil {
			step.ExitCode = upgradeExitCode(err)
			return nil, m.resumeAfterUpgradeFailure(id, err)
		}
		steps = append(steps, step)
	}

	// Phase 5: start.
	emitSection(emit, fmt.Sprintf("starting %s", def.Name))
	status, err := m.Start(id)
	if err != nil {
		return nil, &UpgradeError{Phase: "start", ExitCode: -1, ServiceDown: true, ResumeErr: err, Cause: err}
	}

	return &ServiceUpgradeResult{
		Status:           "ok",
		TmpPath:          tmpPath,
		TargetPath:       targetPath,
		RememberedTarget: remembered,
		Service:          status,
		Steps:            steps,
	}, nil
}

// resumeAfterUpgradeFailure restarts the service after a failure that happened
// while it was stopped, and annotates the error with the outcome.
func (m *Manager) resumeAfterUpgradeFailure(id string, cause error) error {
	var upgradeErr *UpgradeError
	if !errors.As(cause, &upgradeErr) {
		upgradeErr = &UpgradeError{Phase: "post-stop", ExitCode: -1, Cause: cause}
	}
	if _, err := m.Start(id); err != nil {
		upgradeErr.ResumeErr = err
		upgradeErr.ServiceDown = true
		return upgradeErr
	}
	upgradeErr.Resumed = true
	return upgradeErr
}

// upgradeRunner executes upgrade steps for one service.
type upgradeRunner struct {
	manager  *Manager
	def      ServiceDefinition
	timeout  time.Duration
	extraEnv map[string]string
	emit     UpgradeEmitter
}

// run executes one step through `bash -lc 'set -eo pipefail; <command>'` in the
// service working directory and environment. Output is streamed, mirrored into
// the service log, and tailed for error reporting.
func (r *upgradeRunner) run(step ServiceUpgradeStep) error {
	env := mergeUpgradeEnv(buildServiceEnv(r.def), r.extraEnv)

	logFile, err := r.openLog()
	if err != nil {
		return stepError(step, -1, err)
	}
	if logFile != nil {
		defer logFile.Close()
		marker := fmt.Sprintf("\n[%s] %s command %d/%d: %s\n", time.Now().Format(time.RFC3339), step.Phase, step.Index, step.Total, step.Command)
		_, _ = logFile.WriteString(marker)
	}

	workingDir := normalizeWorkingDir(r.def.WorkingDir)
	if err := ensureServiceWorkingDir(workingDir); err != nil {
		return stepError(step, -1, err)
	}

	output := &upgradeStepOutput{logFile: logFile, emit: r.emit}
	cmd := exec.Command("bash", "-lc", buildUpgradeShellCommand(step.Command, env))
	cmd.Dir = workingDir
	cmd.Env = env
	cmd.Stdout = output
	cmd.Stderr = output
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return stepError(step, -1, err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	var timedOut bool
	var waitErr error
	if r.timeout > 0 {
		timer := time.NewTimer(r.timeout)
		select {
		case waitErr = <-done:
			timer.Stop()
		case <-timer.C:
			timedOut = true
			// Kill the whole group: a build's children must not outlive it.
			_ = stopProcessGroup(cmd.Process.Pid)
			waitErr = <-done
		}
	} else {
		waitErr = <-done
	}
	output.Flush()

	if waitErr == nil && !timedOut {
		return nil
	}
	upgradeErr := &UpgradeError{
		Phase:    step.Phase,
		Index:    step.Index,
		Total:    step.Total,
		Command:  step.Command,
		ExitCode: upgradeProcessExitCode(waitErr),
		TimedOut: timedOut,
		Tail:     output.Tail(),
	}
	if timedOut {
		upgradeErr.Cause = fmt.Errorf("step exceeded %s", r.timeout)
	} else {
		upgradeErr.Cause = waitErr
	}
	return upgradeErr
}

// openLog opens the service log for appending so `service logs` explains a
// failed upgrade after the terminal is gone.
func (r *upgradeRunner) openLog() (*os.File, error) {
	logPath := r.manager.serviceLogPath(r.def.ID)
	if strings.TrimSpace(logPath) == "" {
		return nil, nil
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return nil, err
	}
	return os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
}

// upgradeStepOutput forwards step output line by line to the emitter and the
// service log while retaining a bounded tail.
type upgradeStepOutput struct {
	mu      sync.Mutex
	logFile io.Writer
	emit    UpgradeEmitter
	buf     []byte
	tail    []string
}

func (o *upgradeStepOutput) Write(p []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.buf = append(o.buf, p...)
	for {
		index := bytes.IndexByte(o.buf, '\n')
		if index < 0 {
			break
		}
		line := string(o.buf[:index])
		o.buf = o.buf[index+1:]
		o.emitLine(line)
	}
	return len(p), nil
}

// Flush emits any trailing line that arrived without a newline.
func (o *upgradeStepOutput) Flush() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if len(o.buf) == 0 {
		return
	}
	line := string(o.buf)
	o.buf = nil
	o.emitLine(line)
}

func (o *upgradeStepOutput) emitLine(line string) {
	if o.logFile != nil {
		_, _ = io.WriteString(o.logFile, line+"\n")
	}
	o.tail = append(o.tail, line)
	if len(o.tail) > upgradeTailLines {
		o.tail = o.tail[len(o.tail)-upgradeTailLines:]
	}
	if o.emit != nil && strings.TrimSpace(line) != "" {
		o.emit(UpgradeEvent{Kind: UpgradeEventLog, Message: line})
	}
}

// Tail returns the retained output lines, trimmed of blank edges.
func (o *upgradeStepOutput) Tail() []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	lines := append([]string(nil), o.tail...)
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// buildUpgradeShellCommand applies strict mode and restores the resolved PATH,
// which a login shell would otherwise reset (see the service start path).
func buildUpgradeShellCommand(command string, env []string) string {
	parts := []string{"set -eo pipefail"}
	if pathVal := lookupEnvValue(env, "PATH"); pathVal != "" {
		parts = append(parts, "export PATH="+shellQuote(pathVal))
	}
	parts = append(parts, command)
	return strings.Join(parts, "; ")
}

// mergeUpgradeEnv appends extra keys, dropping any base entry they replace so
// the child sees exactly one value per key.
func mergeUpgradeEnv(base []string, extra map[string]string) []string {
	if len(extra) == 0 {
		return base
	}
	merged := make([]string, 0, len(base)+len(extra))
	for _, item := range base {
		key, _, found := strings.Cut(item, "=")
		if found {
			if _, replaced := extra[key]; replaced {
				continue
			}
		}
		merged = append(merged, item)
	}
	for key, value := range extra {
		merged = append(merged, key+"="+value)
	}
	return merged
}

// upgradeStepTimeout resolves the per-step bound: request override, then the
// definition, then the default. An explicit 0 disables the bound.
func upgradeStepTimeout(def ServiceDefinition, req ServiceUpgradeRequest) time.Duration {
	seconds := 0
	configured := false
	if def.UpgradeTimeoutSeconds != nil {
		seconds = *def.UpgradeTimeoutSeconds
		configured = true
	}
	if req.TimeoutSeconds != nil {
		seconds = *req.TimeoutSeconds
		configured = true
	}
	if !configured {
		return DefaultUpgradeStepTimeout
	}
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func upgradeHeader(def ServiceDefinition, preCmds, postCmds []string, targetPath string) string {
	parts := []string{
		fmt.Sprintf("%s (%s)", def.Name, def.ID),
		fmt.Sprintf("%d pre-stop · %d post-stop", len(preCmds), len(postCmds)),
	}
	if strings.TrimSpace(targetPath) != "" {
		parts = append(parts, "target "+targetPath)
	}
	return strings.Join(parts, " · ")
}

func upgradeStepTitle(step ServiceUpgradeStep) string {
	return fmt.Sprintf("%s %d/%d  %s", step.Phase, step.Index, step.Total, step.Command)
}

func stepError(step ServiceUpgradeStep, exitCode int, cause error) error {
	return &UpgradeError{
		Phase:    step.Phase,
		Index:    step.Index,
		Total:    step.Total,
		Command:  step.Command,
		ExitCode: exitCode,
		Cause:    cause,
	}
}

func upgradeExitCode(err error) int {
	var upgradeErr *UpgradeError
	if errors.As(err, &upgradeErr) {
		return upgradeErr.ExitCode
	}
	return -1
}

func upgradeProcessExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}
	return -1
}

func emitSection(emit UpgradeEmitter, title string) {
	if emit == nil {
		return
	}
	emit(UpgradeEvent{Kind: UpgradeEventSection, Message: title})
}

// normalizeUpgradeCmds trims steps and drops blank ones, returning nil for an
// empty result so a cleared list stays cleared.
func normalizeUpgradeCmds(cmds []string) []string {
	if len(cmds) == 0 {
		return nil
	}
	result := make([]string, 0, len(cmds))
	for _, cmd := range cmds {
		if trimmed := strings.TrimSpace(cmd); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func cloneCmdList(cmds []string) []string {
	if len(cmds) == 0 {
		return nil
	}
	return append([]string(nil), cmds...)
}

func cloneIntPtr(value *int) *int {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

// lookupDefinition returns a copy of one service definition.
func (m *Manager) lookupDefinition(id string) (ServiceDefinition, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.findDefinitionLocked(id)
}

// upgradeSummaryLines renders the human-readable rollup the CLI prints as-is,
// so the done frame stays structured data only.
func upgradeSummaryLines(result *ServiceUpgradeResult) []string {
	if result == nil {
		return nil
	}
	lines := make([]string, 0, 4+len(result.Steps))
	for _, step := range result.Steps {
		lines = append(lines, fmt.Sprintf("  ok  %s %d/%d  %s", step.Phase, step.Index, step.Total, step.Command))
	}
	if result.Service != nil {
		lines = append(lines, "", fmt.Sprintf("  Status: %s   PID: %d", result.Service.Status, result.Service.PID))
	}
	return lines
}
