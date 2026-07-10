// Package crontasks manages scheduled shell commands (interval or 5-field UTC cron).
package crontasks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/xhd2015/agent-pro/agent/exec/tool_resolve"
	"github.com/xhd2015/ai-critic/server/config"
)

const (
	StatusIdle    = "idle"
	StatusRunning = "running"
	StatusError   = "error"

	ScheduleInterval = "interval"
	ScheduleCron     = "cron"

	defaultTimeout = "1h"
	historyRetain  = 7 * 24 * time.Hour
	tickInterval   = 1 * time.Second
)

// CronTaskDefinition is one row in cron-tasks.json.
type CronTaskDefinition struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Command      string            `json:"command"`
	WorkingDir   string            `json:"workingDir,omitempty"`
	ExtraEnv     map[string]string `json:"extraEnv,omitempty"`
	ScheduleMode string            `json:"scheduleMode"` // interval | cron
	Interval     string            `json:"interval,omitempty"`
	CronExpr     string            `json:"cronExpr,omitempty"`
	Enabled      *bool             `json:"enabled,omitempty"`
	Timeout      string            `json:"timeout,omitempty"`
	CreatedAt    string            `json:"createdAt"`
	UpdatedAt    string            `json:"updatedAt"`
	RecentRuns   []CronTaskRun     `json:"recentRuns,omitempty"`
}

// CronTaskRun is one execution record (history, 7 days).
type CronTaskRun struct {
	StartedAt  string `json:"startedAt"`
	FinishedAt string `json:"finishedAt,omitempty"`
	ExitCode   *int   `json:"exitCode,omitempty"`
	Error      string `json:"error,omitempty"`
}

// CronTaskStatus is the list/create API view.
type CronTaskStatus struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Command        string            `json:"command"`
	WorkingDir     string            `json:"workingDir,omitempty"`
	ExtraEnv       map[string]string `json:"extraEnv,omitempty"`
	ScheduleMode   string            `json:"scheduleMode"`
	Interval       string            `json:"interval,omitempty"`
	CronExpr       string            `json:"cronExpr,omitempty"`
	Enabled        bool              `json:"enabled"`
	Timeout        string            `json:"timeout,omitempty"`
	Status         string            `json:"status"`
	PID            int               `json:"pid,omitempty"`
	LastStartedAt  string            `json:"lastStartedAt,omitempty"`
	LastFinishedAt string            `json:"lastFinishedAt,omitempty"`
	LastExitCode   *int              `json:"lastExitCode,omitempty"`
	LastError      string            `json:"lastError,omitempty"`
	NextRunAt      string            `json:"nextRunAt,omitempty"`
	LogPath        string            `json:"logPath"`
	RecentRuns     []CronTaskRun     `json:"recentRuns,omitempty"`
	CreatedAt      string            `json:"createdAt,omitempty"`
	UpdatedAt      string            `json:"updatedAt,omitempty"`
}

type taskRuntime struct {
	cmd              *exec.Cmd
	pid              int
	runID            uint64 // increments each start; wait/timeout ignore stale runs
	status           string
	lastStartedAt    string
	lastFinishedAt   string
	lastExitCode     *int
	lastError        string
	cancelTimeout    chan struct{}
	killOnce         sync.Once
	suppressSchedule bool // set on timeout; cleared by enable / manual run
}

// Manager owns definitions and the tick loop.
type Manager struct {
	mu        sync.Mutex
	defs      []CronTaskDefinition
	runtime   map[string]*taskRuntime
	tickStop  chan struct{}
	started   bool
	configPath string
}

var (
	defaultManager = NewManager()
)

func NewManager() *Manager {
	m := &Manager{
		runtime:    make(map[string]*taskRuntime),
		configPath: filepath.Join(config.DataDir, "cron-tasks.json"),
	}
	m.loadLocked()
	return m
}

func GetDefaultManager() *Manager {
	return defaultManager
}

// RegisterAPI mounts /api/cron-tasks* routes.
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/cron-tasks", handleCronTasks)
	mux.HandleFunc("/api/cron-tasks/enable", handleEnable)
	mux.HandleFunc("/api/cron-tasks/disable", handleDisable)
	mux.HandleFunc("/api/cron-tasks/run", handleRun)
	mux.HandleFunc("/api/cron-tasks/history", handleHistory)
}

// Start begins the 1s scheduler tick.
func Start() {
	defaultManager.Start()
}

// Shutdown stops the tick loop and kills running tasks.
func Shutdown() {
	defaultManager.Shutdown()
}

func (m *Manager) Start() {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return
	}
	m.started = true
	m.tickStop = make(chan struct{})
	stopCh := m.tickStop
	// Prune seed history on boot.
	changed := false
	for i := range m.defs {
		if pruneRuns(&m.defs[i]) {
			changed = true
		}
	}
	if changed {
		_ = m.saveLocked()
	}
	m.mu.Unlock()

	go func() {
		ticker := time.NewTicker(tickInterval)
		defer ticker.Stop()
		// Fire due tasks promptly after start.
		m.tick()
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				m.tick()
			}
		}
	}()
}

func (m *Manager) Shutdown() {
	m.mu.Lock()
	if m.started && m.tickStop != nil {
		close(m.tickStop)
		m.started = false
		m.tickStop = nil
	}
	ids := make([]string, 0, len(m.runtime))
	for id, rt := range m.runtime {
		if rt != nil && rt.pid > 0 {
			ids = append(ids, id)
		}
	}
	m.mu.Unlock()
	for _, id := range ids {
		m.killTask(id, "shutdown") // no runID → kill current
	}
}

func (m *Manager) loadLocked() {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Printf("[cron-tasks] failed to read config: %v\n", err)
		}
		m.defs = []CronTaskDefinition{}
		return
	}
	var defs []CronTaskDefinition
	if err := json.Unmarshal(data, &defs); err != nil {
		fmt.Printf("[cron-tasks] failed to parse config: %v\n", err)
		m.defs = []CronTaskDefinition{}
		return
	}
	for i := range defs {
		pruneRuns(&defs[i])
		if strings.TrimSpace(defs[i].Timeout) == "" {
			defs[i].Timeout = defaultTimeout
		}
	}
	m.defs = defs
}

func (m *Manager) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(m.configPath), 0755); err != nil {
		return err
	}
	// Ensure DataDir/cron-tasks exists for logs too.
	_ = os.MkdirAll(filepath.Join(config.DataDir, "cron-tasks"), 0755)
	data, err := json.MarshalIndent(m.defs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.configPath, data, 0644)
}

func (m *Manager) List() []CronTaskStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]CronTaskStatus, 0, len(m.defs))
	now := time.Now().UTC()
	for _, def := range m.defs {
		out = append(out, m.buildStatusLocked(def, now))
	}
	return out
}

func (m *Manager) Get(id string) (CronTaskStatus, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	def, ok := m.findDefLocked(id)
	if !ok {
		return CronTaskStatus{}, false
	}
	return m.buildStatusLocked(def, time.Now().UTC()), true
}

func (m *Manager) History(id string) ([]CronTaskRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	def, ok := m.findDefLocked(id)
	if !ok {
		// Also resolve by name.
		def, ok = m.findDefByNameLocked(id)
		if !ok {
			return nil, fmt.Errorf("cron task not found")
		}
	}
	if pruneRuns(&def) {
		for i := range m.defs {
			if m.defs[i].ID == def.ID {
				m.defs[i].RecentRuns = def.RecentRuns
				break
			}
		}
		_ = m.saveLocked()
	}
	runs := append([]CronTaskRun(nil), def.RecentRuns...)
	return runs, nil
}

func (m *Manager) Create(req CronTaskDefinition) (CronTaskStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := validateDefinition(&req, true); err != nil {
		return CronTaskStatus{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if req.ID == "" {
		req.ID = fmt.Sprintf("cron-%d", time.Now().UnixNano())
	}
	if _, exists := m.findDefLocked(req.ID); exists {
		return CronTaskStatus{}, fmt.Errorf("cron task id already exists")
	}
	for _, d := range m.defs {
		if d.Name == req.Name {
			return CronTaskStatus{}, fmt.Errorf("cron task name already exists")
		}
	}
	req.CreatedAt = now
	req.UpdatedAt = now
	if strings.TrimSpace(req.Timeout) == "" {
		req.Timeout = defaultTimeout
	}
	if req.RecentRuns == nil {
		req.RecentRuns = []CronTaskRun{}
	}
	pruneRuns(&req)
	m.defs = append(m.defs, req)
	if err := m.saveLocked(); err != nil {
		m.defs = m.defs[:len(m.defs)-1]
		return CronTaskStatus{}, err
	}
	return m.buildStatusLocked(req, time.Now().UTC()), nil
}

func (m *Manager) Update(req CronTaskDefinition) (CronTaskStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if req.ID == "" {
		return CronTaskStatus{}, fmt.Errorf("id is required")
	}
	idx := -1
	for i, d := range m.defs {
		if d.ID == req.ID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return CronTaskStatus{}, fmt.Errorf("cron task not found")
	}
	// Merge: keep existing fields when partial? Tests use full create-like body.
	// For update API, apply non-empty fields from req onto existing.
	cur := m.defs[idx]
	merged := cur
	if req.Name != "" {
		merged.Name = req.Name
	}
	if req.Command != "" {
		merged.Command = req.Command
	}
	if req.WorkingDir != "" || req.WorkingDir == "" && req.Name != "" {
		// only set if provided in JSON via full replace of known fields
	}
	if req.WorkingDir != "" {
		merged.WorkingDir = req.WorkingDir
	}
	if req.ExtraEnv != nil {
		merged.ExtraEnv = req.ExtraEnv
	}
	if req.ScheduleMode != "" {
		merged.ScheduleMode = req.ScheduleMode
	}
	if req.Interval != "" {
		merged.Interval = req.Interval
	}
	if req.CronExpr != "" {
		merged.CronExpr = req.CronExpr
	}
	// When switching modes, clear the other schedule field.
	if merged.ScheduleMode == ScheduleInterval {
		merged.CronExpr = ""
	}
	if merged.ScheduleMode == ScheduleCron {
		merged.Interval = ""
	}
	if req.Timeout != "" {
		merged.Timeout = req.Timeout
	}
	if req.Enabled != nil {
		merged.Enabled = req.Enabled
	}
	if err := validateDefinition(&merged, false); err != nil {
		return CronTaskStatus{}, err
	}
	if strings.TrimSpace(merged.Timeout) == "" {
		merged.Timeout = defaultTimeout
	}
	merged.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	// Preserve history and createdAt
	merged.RecentRuns = cur.RecentRuns
	merged.CreatedAt = cur.CreatedAt
	pruneRuns(&merged)
	m.defs[idx] = merged
	if err := m.saveLocked(); err != nil {
		return CronTaskStatus{}, err
	}
	return m.buildStatusLocked(merged, time.Now().UTC()), nil
}

func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	idx := -1
	for i, d := range m.defs {
		if d.ID == id || d.Name == id {
			idx = i
			id = d.ID
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("cron task not found")
	}
	// Stop if running
	if rt := m.runtime[id]; rt != nil && rt.pid > 0 {
		m.mu.Unlock()
		m.killTask(id, "deleted") // no runID → kill current
		m.mu.Lock()
	}
	m.defs = append(m.defs[:idx], m.defs[idx+1:]...)
	delete(m.runtime, id)
	return m.saveLocked()
}

func (m *Manager) SetEnabled(id string, enabled bool) (CronTaskStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	idx := -1
	for i, d := range m.defs {
		if d.ID == id || d.Name == id {
			idx = i
			id = d.ID
			break
		}
	}
	if idx < 0 {
		return CronTaskStatus{}, fmt.Errorf("cron task not found")
	}
	v := enabled
	m.defs[idx].Enabled = &v
	m.defs[idx].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	// Re-enable clears timeout hold so the schedule can fire again.
	if enabled {
		if rt := m.runtime[id]; rt != nil {
			rt.suppressSchedule = false
		}
	}
	if err := m.saveLocked(); err != nil {
		return CronTaskStatus{}, err
	}
	return m.buildStatusLocked(m.defs[idx], time.Now().UTC()), nil
}

// RunNow starts a task immediately if not already running (skip on overlap).
func (m *Manager) RunNow(id string) (CronTaskStatus, error) {
	m.mu.Lock()
	def, ok := m.findDefLocked(id)
	if !ok {
		def, ok = m.findDefByNameLocked(id)
	}
	if !ok {
		m.mu.Unlock()
		return CronTaskStatus{}, fmt.Errorf("cron task not found")
	}
	id = def.ID
	if m.isRunningLocked(id) {
		st := m.buildStatusLocked(def, time.Now().UTC())
		m.mu.Unlock()
		return st, nil // skip
	}
	// Manual run clears timeout hold.
	if rt := m.runtime[id]; rt != nil {
		rt.suppressSchedule = false
	}
	m.mu.Unlock()
	if err := m.startTask(id); err != nil {
		return CronTaskStatus{}, err
	}
	st, _ := m.Get(id)
	return st, nil
}

func (m *Manager) tick() {
	m.mu.Lock()
	now := time.Now().UTC()
	type dueItem struct {
		id string
	}
	var due []dueItem
	changed := false
	for i := range m.defs {
		if pruneRuns(&m.defs[i]) {
			changed = true
		}
		def := m.defs[i]
		if !taskEnabled(def) {
			continue
		}
		if m.isRunningLocked(def.ID) {
			continue // overlap = skip
		}
		// After a timeout kill, hold auto-schedule until enable or manual run.
		if rt := m.runtime[def.ID]; rt != nil && rt.suppressSchedule {
			continue
		}
		if m.isDueLocked(def, now) {
			due = append(due, dueItem{id: def.ID})
		}
	}
	if changed {
		_ = m.saveLocked()
	}
	m.mu.Unlock()

	for _, d := range due {
		if err := m.startTask(d.id); err != nil {
			fmt.Printf("[cron-tasks] failed to start %s: %v\n", d.id, err)
		}
	}
}

func (m *Manager) isDueLocked(def CronTaskDefinition, now time.Time) bool {
	rt := m.runtime[def.ID]
	lastFinish := ""
	if rt != nil {
		lastFinish = rt.lastFinishedAt
	}
	// Also consider last finished from history if runtime empty (after restart).
	if lastFinish == "" && len(def.RecentRuns) > 0 {
		last := def.RecentRuns[len(def.RecentRuns)-1]
		if last.FinishedAt != "" {
			lastFinish = last.FinishedAt
		} else if last.StartedAt != "" && (rt == nil || rt.pid == 0) {
			// unfinished run recorded but process gone — treat finished at start
			lastFinish = last.StartedAt
		}
	}

	switch def.ScheduleMode {
	case ScheduleInterval:
		iv, err := time.ParseDuration(strings.TrimSpace(def.Interval))
		if err != nil || iv <= 0 {
			return false
		}
		if lastFinish == "" {
			// No prior finish → due promptly after create/enable.
			return true
		}
		ft, err := time.Parse(time.RFC3339, lastFinish)
		if err != nil {
			return true
		}
		return !now.Before(ft.Add(iv))
	case ScheduleCron:
		next, err := nextCronUTC(def.CronExpr, now.Add(-time.Second))
		if err != nil {
			return false
		}
		// Due if next fire is at or before now (within this second).
		return !next.After(now)
	default:
		return false
	}
}

func (m *Manager) isRunningLocked(id string) bool {
	rt := m.runtime[id]
	if rt == nil || rt.pid <= 0 {
		return false
	}
	if !processAlive(rt.pid) {
		return false
	}
	return rt.status == StatusRunning
}

func (m *Manager) buildStatusLocked(def CronTaskDefinition, now time.Time) CronTaskStatus {
	rt := m.runtime[def.ID]
	st := CronTaskStatus{
		ID:           def.ID,
		Name:         def.Name,
		Command:      def.Command,
		WorkingDir:   def.WorkingDir,
		ExtraEnv:     def.ExtraEnv,
		ScheduleMode: def.ScheduleMode,
		Interval:     def.Interval,
		CronExpr:     def.CronExpr,
		Enabled:      taskEnabled(def),
		Timeout:      def.Timeout,
		Status:       StatusIdle,
		LogPath:      taskLogPath(def.ID),
		RecentRuns:   append([]CronTaskRun(nil), def.RecentRuns...),
		CreatedAt:    def.CreatedAt,
		UpdatedAt:    def.UpdatedAt,
	}
	if st.Timeout == "" {
		st.Timeout = defaultTimeout
	}
	if rt != nil {
		st.LastStartedAt = rt.lastStartedAt
		st.LastFinishedAt = rt.lastFinishedAt
		st.LastExitCode = rt.lastExitCode
		st.LastError = rt.lastError
		if rt.pid > 0 && processAlive(rt.pid) && rt.status == StatusRunning {
			st.Status = StatusRunning
			st.PID = rt.pid
		} else if rt.lastError != "" {
			st.Status = StatusError
		}
	} else if len(def.RecentRuns) > 0 {
		last := def.RecentRuns[len(def.RecentRuns)-1]
		st.LastStartedAt = last.StartedAt
		st.LastFinishedAt = last.FinishedAt
		st.LastExitCode = last.ExitCode
		st.LastError = last.Error
		if last.Error != "" {
			st.Status = StatusError
		}
	}

	// Next run
	if taskEnabled(def) && st.Status != StatusRunning {
		if next := m.computeNextRunLocked(def, now, st.LastFinishedAt); !next.IsZero() {
			st.NextRunAt = next.UTC().Format(time.RFC3339)
		}
	} else if st.Status == StatusRunning && def.ScheduleMode == ScheduleInterval {
		// After this finish, next = finish+interval; estimate if we have no finish yet.
		// Prefer leave empty while running, or compute from last start + 0.
	} else if st.Status == StatusRunning && st.LastFinishedAt != "" {
		// keep previous finish-based next if any
	}
	// When running under interval mode, next is after current finish.
	if st.Status == StatusRunning && def.ScheduleMode == ScheduleInterval {
		// not yet known; leave empty or set loosely — tests check after finish
	}
	// After finished (idle), ensure next is finish+interval
	if st.Status != StatusRunning && def.ScheduleMode == ScheduleInterval && st.LastFinishedAt != "" {
		if iv, err := time.ParseDuration(def.Interval); err == nil {
			if ft, err := time.Parse(time.RFC3339, st.LastFinishedAt); err == nil {
				st.NextRunAt = ft.Add(iv).UTC().Format(time.RFC3339)
			}
		}
	}

	return st
}

func (m *Manager) computeNextRunLocked(def CronTaskDefinition, now time.Time, lastFinished string) time.Time {
	switch def.ScheduleMode {
	case ScheduleInterval:
		iv, err := time.ParseDuration(strings.TrimSpace(def.Interval))
		if err != nil || iv <= 0 {
			return time.Time{}
		}
		if lastFinished == "" {
			return now // due now
		}
		ft, err := time.Parse(time.RFC3339, lastFinished)
		if err != nil {
			return now
		}
		return ft.Add(iv)
	case ScheduleCron:
		next, err := nextCronUTC(def.CronExpr, now)
		if err != nil {
			return time.Time{}
		}
		return next
	default:
		return time.Time{}
	}
}

func (m *Manager) findDefLocked(id string) (CronTaskDefinition, bool) {
	for _, d := range m.defs {
		if d.ID == id {
			return d, true
		}
	}
	return CronTaskDefinition{}, false
}

func (m *Manager) findDefByNameLocked(name string) (CronTaskDefinition, bool) {
	for _, d := range m.defs {
		if d.Name == name {
			return d, true
		}
	}
	return CronTaskDefinition{}, false
}

func taskEnabled(def CronTaskDefinition) bool {
	if def.Enabled == nil {
		return true
	}
	return *def.Enabled
}

func taskLogPath(id string) string {
	return filepath.Join(config.DataDir, "cron-tasks", id+".log")
}

func validateDefinition(def *CronTaskDefinition, creating bool) error {
	if strings.TrimSpace(def.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(def.Command) == "" {
		return fmt.Errorf("command is required")
	}

	hasInterval := strings.TrimSpace(def.Interval) != ""
	hasCron := strings.TrimSpace(def.CronExpr) != ""
	mode := strings.TrimSpace(def.ScheduleMode)

	if hasInterval && hasCron {
		return fmt.Errorf("schedule: specify either interval or cron, not both")
	}
	if mode == "" && !hasInterval && !hasCron {
		return fmt.Errorf("schedule is required: provide scheduleMode with interval or cronExpr")
	}
	if mode == "" {
		if hasInterval {
			mode = ScheduleInterval
			def.ScheduleMode = ScheduleInterval
		} else if hasCron {
			mode = ScheduleCron
			def.ScheduleMode = ScheduleCron
		}
	}
	switch mode {
	case ScheduleInterval:
		if !hasInterval {
			return fmt.Errorf("interval is required for scheduleMode=interval")
		}
		if hasCron {
			return fmt.Errorf("schedule: interval mode cannot include cronExpr")
		}
		if _, err := time.ParseDuration(strings.TrimSpace(def.Interval)); err != nil {
			return fmt.Errorf("invalid interval: %w", err)
		}
		def.CronExpr = ""
	case ScheduleCron:
		if !hasCron {
			return fmt.Errorf("cronExpr is required for scheduleMode=cron")
		}
		if hasInterval {
			return fmt.Errorf("schedule: cron mode cannot include interval")
		}
		if err := validateCronExpr(def.CronExpr); err != nil {
			return fmt.Errorf("invalid cronExpr: %w", err)
		}
		def.Interval = ""
	default:
		return fmt.Errorf("scheduleMode must be %q or %q", ScheduleInterval, ScheduleCron)
	}

	// Timeout: empty → default; must be > 0 when set
	to := strings.TrimSpace(def.Timeout)
	if to == "" {
		if creating {
			def.Timeout = defaultTimeout
		}
	} else {
		d, err := time.ParseDuration(to)
		if err != nil {
			return fmt.Errorf("invalid timeout: %w", err)
		}
		if d <= 0 {
			return fmt.Errorf("timeout must be > 0 (got %q); unlimited is not allowed", to)
		}
	}
	return nil
}

func pruneRuns(def *CronTaskDefinition) bool {
	if len(def.RecentRuns) == 0 {
		return false
	}
	cutoff := time.Now().UTC().Add(-historyRetain)
	kept := def.RecentRuns[:0]
	changed := false
	for _, r := range def.RecentRuns {
		ts := r.StartedAt
		if ts == "" {
			ts = r.FinishedAt
		}
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			// keep unparseable
			kept = append(kept, r)
			continue
		}
		if t.Before(cutoff) {
			changed = true
			continue
		}
		kept = append(kept, r)
	}
	if changed {
		def.RecentRuns = kept
	}
	return changed
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

// --- process execution ---

func (m *Manager) startTask(id string) error {
	m.mu.Lock()
	def, ok := m.findDefLocked(id)
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("cron task not found")
	}
	if m.isRunningLocked(id) {
		m.mu.Unlock()
		return nil // skip overlap
	}
	timeoutStr := def.Timeout
	if timeoutStr == "" {
		timeoutStr = defaultTimeout
	}
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil || timeout <= 0 {
		m.mu.Unlock()
		return fmt.Errorf("invalid timeout %q", timeoutStr)
	}
	logPath := taskLogPath(id)
	workingDir := strings.TrimSpace(def.WorkingDir)
	command := def.Command
	extraEnv := def.ExtraEnv
	name := def.Name

	// Prepare runtime shell before unlock for status visibility
	rt := m.runtime[id]
	if rt == nil {
		rt = &taskRuntime{}
		m.runtime[id] = rt
	}
	if rt.suppressSchedule {
		// Auto-start while held after timeout — refuse (manual/enable clears hold first).
		m.mu.Unlock()
		return nil
	}
	rt.status = StatusRunning
	// Keep prior lastError until the process is actually started so list/history
	// still surface "timeout" if Start races with status polls.
	rt.cancelTimeout = make(chan struct{})
	rt.killOnce = sync.Once{}
	rt.runID++
	runID := rt.runID
	cancelCh := rt.cancelTimeout
	m.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		m.recordStartFailure(id, err.Error())
		return err
	}
	if workingDir != "" {
		if err := os.MkdirAll(workingDir, 0755); err != nil {
			m.recordStartFailure(id, err.Error())
			return err
		}
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		m.recordStartFailure(id, err.Error())
		return err
	}

	startedAt := time.Now().UTC()
	_, _ = logFile.WriteString(fmt.Sprintf("\n[%s] starting cron task %s\n", startedAt.Format(time.RFC3339), name))

	env := tool_resolve.AppendExtraPaths(os.Environ())
	if len(extraEnv) > 0 {
		for k, v := range extraEnv {
			env = append(env, k+"="+v)
		}
	}
	shellCommand := command
	if pathVal := lookupEnv(env, "PATH"); pathVal != "" {
		shellCommand = fmt.Sprintf("export PATH=%s; %s", shellQuote(pathVal), command)
	}
	cmd := exec.Command("bash", "-lc", shellCommand)
	if workingDir != "" {
		cmd.Dir = workingDir
	}
	cmd.Env = env
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		_, _ = logFile.WriteString(fmt.Sprintf("[%s] failed to start: %v\n", time.Now().Format(time.RFC3339), err))
		_ = logFile.Close()
		m.recordStartFailure(id, err.Error())
		return err
	}

	pid := cmd.Process.Pid
	startedStr := startedAt.Format(time.RFC3339)

	m.mu.Lock()
	rt = m.runtime[id]
	if rt == nil {
		rt = &taskRuntime{}
		m.runtime[id] = rt
	}
	// Aborted / superseded before Start completed.
	if rt.runID != runID {
		m.mu.Unlock()
		_ = stopProcessGroup(pid)
		_ = logFile.Close()
		return nil
	}
	rt.cmd = cmd
	rt.pid = pid
	rt.status = StatusRunning
	rt.lastStartedAt = startedStr
	rt.lastError = ""
	// Record history start
	for i := range m.defs {
		if m.defs[i].ID == id {
			m.defs[i].RecentRuns = append(m.defs[i].RecentRuns, CronTaskRun{
				StartedAt: startedStr,
			})
			pruneRuns(&m.defs[i])
			_ = m.saveLocked()
			break
		}
	}
	m.mu.Unlock()

	// Timeout enforcer — keyed by runID so a stale timer cannot kill a later run.
	go func(runID uint64, cancelCh chan struct{}) {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		select {
		case <-timer.C:
			m.killTask(id, "timeout", runID)
		case <-cancelCh:
		}
	}(runID, cancelCh)

	go m.waitForExit(id, cmd, logFile, startedStr, runID)
	return nil
}

func (m *Manager) waitForExit(id string, cmd *exec.Cmd, logFile *os.File, startedStr string, runID uint64) {
	err := cmd.Wait()
	finishedAt := time.Now().UTC()
	finishedStr := finishedAt.Format(time.RFC3339)

	exitCode := 0
	errMsg := ""
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			exitCode = -1
			errMsg = err.Error()
		}
		_, _ = logFile.WriteString(fmt.Sprintf("[%s] exited with error: %v\n", finishedStr, err))
	} else {
		_, _ = logFile.WriteString(fmt.Sprintf("[%s] exited ok\n", finishedStr))
	}
	_ = logFile.Close()

	m.mu.Lock()
	defer m.mu.Unlock()
	rt := m.runtime[id]
	if rt == nil || rt.runID != runID {
		return
	}
	// If kill set timeout error already, preserve it.
	if rt.lastError == "timeout" || strings.Contains(rt.lastError, "timeout") {
		errMsg = rt.lastError
	} else if errMsg == "" && rt.lastError != "" {
		errMsg = rt.lastError
	}
	if cancel := rt.cancelTimeout; cancel != nil {
		select {
		case <-cancel:
		default:
			close(cancel)
		}
	}
	rt.cmd = nil
	rt.pid = 0
	rt.lastFinishedAt = finishedStr
	code := exitCode
	rt.lastExitCode = &code
	if errMsg != "" {
		rt.lastError = errMsg
		rt.status = StatusError
	} else {
		rt.lastError = ""
		rt.status = StatusIdle
	}

	// Update history last matching startedAt
	for i := range m.defs {
		if m.defs[i].ID != id {
			continue
		}
		runs := m.defs[i].RecentRuns
		updated := false
		for j := len(runs) - 1; j >= 0; j-- {
			if runs[j].StartedAt == startedStr && runs[j].FinishedAt == "" {
				runs[j].FinishedAt = finishedStr
				runs[j].ExitCode = &code
				if errMsg != "" {
					runs[j].Error = errMsg
				}
				updated = true
				break
			}
		}
		if !updated {
			r := CronTaskRun{
				StartedAt:  startedStr,
				FinishedAt: finishedStr,
				ExitCode:   &code,
				Error:      errMsg,
			}
			m.defs[i].RecentRuns = append(m.defs[i].RecentRuns, r)
		} else {
			m.defs[i].RecentRuns = runs
		}
		pruneRuns(&m.defs[i])
		_ = m.saveLocked()
		break
	}
}

func (m *Manager) killTask(id string, reason string, runIDs ...uint64) {
	var requiredRunID uint64
	hasRunID := len(runIDs) > 0
	if hasRunID {
		requiredRunID = runIDs[0]
	}

	m.mu.Lock()
	rt := m.runtime[id]
	if rt == nil || rt.pid <= 0 {
		m.mu.Unlock()
		return
	}
	if hasRunID && rt.runID != requiredRunID {
		m.mu.Unlock()
		return
	}
	pid := rt.pid
	// Mark error before kill so waitForExit sees it
	if reason != "" {
		if reason == "timeout" {
			rt.lastError = "timeout"
			// Hold auto-schedule after timeout so a short-interval task does not
			// immediately respawn and look "still alive" past the deadline.
			rt.suppressSchedule = true
		} else {
			rt.lastError = reason
		}
	}
	var doKill bool
	rt.killOnce.Do(func() { doKill = true })
	m.mu.Unlock()

	if doKill {
		_ = stopProcessGroup(pid)
	}

	// Ensure lastError sticks even if process already exiting
	m.mu.Lock()
	if rt2 := m.runtime[id]; rt2 != nil {
		if reason == "timeout" && (!hasRunID || rt2.runID == requiredRunID) {
			rt2.lastError = "timeout"
			rt2.suppressSchedule = true
		}
	}
	m.mu.Unlock()
}

func (m *Manager) recordStartFailure(id string, msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rt := m.runtime[id]
	if rt == nil {
		rt = &taskRuntime{}
		m.runtime[id] = rt
	}
	rt.status = StatusError
	rt.pid = 0
	rt.cmd = nil
	rt.lastError = msg
	now := time.Now().UTC().Format(time.RFC3339)
	rt.lastStartedAt = now
	rt.lastFinishedAt = now
	code := -1
	rt.lastExitCode = &code
	for i := range m.defs {
		if m.defs[i].ID == id {
			m.defs[i].RecentRuns = append(m.defs[i].RecentRuns, CronTaskRun{
				StartedAt:  now,
				FinishedAt: now,
				ExitCode:   &code,
				Error:      msg,
			})
			pruneRuns(&m.defs[i])
			_ = m.saveLocked()
			break
		}
	}
}

func stopProcessGroup(pid int) error {
	if pid <= 0 {
		return nil
	}
	pgid, err := syscall.Getpgid(pid)
	target := pid
	if err == nil {
		target = pgid
	}
	// Graceful first (process group), then escalate quickly so timeout windows
	// in tests (and production) actually free the PID within a second or two.
	_ = syscall.Kill(-target, syscall.SIGTERM)
	_ = syscall.Kill(pid, syscall.SIGTERM)
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	_ = syscall.Kill(-target, syscall.SIGKILL)
	_ = syscall.Kill(pid, syscall.SIGKILL)
	// Also SIGKILL the process group again in case children reparented slowly.
	if target != pid {
		_ = syscall.Kill(-target, syscall.SIGKILL)
	}
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return fmt.Errorf("process %d did not exit", pid)
}

func lookupEnv(env []string, key string) string {
	prefix := key + "="
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			return strings.TrimPrefix(item, prefix)
		}
	}
	return ""
}

func shellQuote(s string) string {
	safe := true
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '/' || c == '.' || c == '-' || c == '_') {
			safe = false
			break
		}
	}
	if safe && s != "" {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// --- HTTP ---

func handleCronTasks(w http.ResponseWriter, r *http.Request) {
	mgr := GetDefaultManager()
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, mgr.List())
	case http.MethodPost:
		var req CronTaskDefinition
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		st, err := mgr.Create(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, st)
	case http.MethodPut:
		var req CronTaskDefinition
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		st, err := mgr.Update(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, st)
	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id is required", http.StatusBadRequest)
			return
		}
		if err := mgr.Delete(id); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleEnable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	st, err := GetDefaultManager().SetEnabled(id, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func handleDisable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	st, err := GetDefaultManager().SetEnabled(id, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	st, err := GetDefaultManager().RunNow(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	runs, err := GetDefaultManager().History(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if runs == nil {
		runs = []CronTaskRun{}
	}
	writeJSON(w, http.StatusOK, runs)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// --- cron expression (5-field UTC) ---

func validateCronExpr(expr string) error {
	fields := strings.Fields(strings.TrimSpace(expr))
	if len(fields) != 5 {
		return fmt.Errorf("want 5 fields, got %d", len(fields))
	}
	return nil
}

// nextCronUTC returns the next time at or after `from` (UTC) matching the expr.
// Supports: *, N, N-M, */step, N-M/step, lists of those. No names.
func nextCronUTC(expr string, from time.Time) (time.Time, error) {
	fields := strings.Fields(strings.TrimSpace(expr))
	if len(fields) != 5 {
		return time.Time{}, fmt.Errorf("want 5 fields")
	}
	minuteF, err := parseCronField(fields[0], 0, 59)
	if err != nil {
		return time.Time{}, err
	}
	hourF, err := parseCronField(fields[1], 0, 23)
	if err != nil {
		return time.Time{}, err
	}
	domF, err := parseCronField(fields[2], 1, 31)
	if err != nil {
		return time.Time{}, err
	}
	monthF, err := parseCronField(fields[3], 1, 12)
	if err != nil {
		return time.Time{}, err
	}
	dowF, err := parseCronField(fields[4], 0, 6)
	if err != nil {
		return time.Time{}, err
	}

	// Search minute by minute up to ~2 years.
	t := from.UTC().Truncate(time.Minute).Add(time.Minute)
	limit := t.Add(2 * 365 * 24 * time.Hour)
	for t.Before(limit) {
		if monthF[int(t.Month())] &&
			minuteF[t.Minute()] &&
			hourF[t.Hour()] &&
			domF[t.Day()] &&
			dowF[int(t.Weekday())] {
			return t, nil
		}
		t = t.Add(time.Minute)
	}
	return time.Time{}, fmt.Errorf("no matching time")
}

func parseCronField(field string, min, max int) (map[int]bool, error) {
	out := make(map[int]bool)
	for _, part := range strings.Split(field, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		step := 1
		rangePart := part
		if strings.Contains(part, "/") {
			bits := strings.SplitN(part, "/", 2)
			rangePart = bits[0]
			s, err := strconv.Atoi(bits[1])
			if err != nil || s <= 0 {
				return nil, fmt.Errorf("invalid step in %q", field)
			}
			step = s
		}
		var start, end int
		if rangePart == "*" {
			start, end = min, max
		} else if strings.Contains(rangePart, "-") {
			bits := strings.SplitN(rangePart, "-", 2)
			var err error
			start, err = strconv.Atoi(bits[0])
			if err != nil {
				return nil, err
			}
			end, err = strconv.Atoi(bits[1])
			if err != nil {
				return nil, err
			}
		} else {
			v, err := strconv.Atoi(rangePart)
			if err != nil {
				return nil, fmt.Errorf("invalid cron field %q", field)
			}
			start, end = v, v
		}
		if start < min {
			start = min
		}
		if end > max {
			end = max
		}
		for v := start; v <= end; v += step {
			out[v] = true
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty cron field %q", field)
	}
	return out, nil
}
