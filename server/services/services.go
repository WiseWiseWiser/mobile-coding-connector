package services

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

	"github.com/xhd2015/lifelog-private/ai-critic/server/cloudflare"
	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
	"github.com/xhd2015/lifelog-private/ai-critic/server/proxy/portforward"
	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_resolve"
)

const (
	StatusStarting = "starting"
	StatusRunning  = "running"
	StatusStopped  = "stopped"
	StatusError    = "error"
)

const (
	restartBackoffBase   = 1 * time.Second
	restartBackoffMax    = 5 * time.Minute
	stableRunResetWindow = 30 * time.Second
)

type ServicePortForward struct {
	Port       int    `json:"port"`
	Label      string `json:"label,omitempty"`
	Provider   string `json:"provider,omitempty"`
	BaseDomain string `json:"baseDomain,omitempty"`
	Subdomain  string `json:"subdomain,omitempty"`
}

type ServiceDefinition struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Command     string              `json:"command"`
	ProjectDir  string              `json:"projectDir,omitempty"`
	WorkingDir  string              `json:"workingDir,omitempty"`
	ExtraEnv    map[string]string   `json:"extraEnv,omitempty"`
	PortForward *ServicePortForward `json:"portForward,omitempty"`
	CreatedAt   string              `json:"createdAt"`
	UpdatedAt   string              `json:"updatedAt"`
}

type ServicePortForwardStatus struct {
	Port       int    `json:"port"`
	Label      string `json:"label,omitempty"`
	Provider   string `json:"provider,omitempty"`
	BaseDomain string `json:"baseDomain,omitempty"`
	Subdomain  string `json:"subdomain,omitempty"`
	PublicURL  string `json:"publicUrl,omitempty"`
	Status     string `json:"status,omitempty"`
	Error      string `json:"error,omitempty"`
	Active     bool   `json:"active"`
}

type ServiceStatus struct {
	ID             string                    `json:"id"`
	Name           string                    `json:"name"`
	Command        string                    `json:"command"`
	ProjectDir     string                    `json:"projectDir,omitempty"`
	WorkingDir     string                    `json:"workingDir,omitempty"`
	ExtraEnv       map[string]string         `json:"extraEnv,omitempty"`
	EffectivePath  string                    `json:"effectivePath,omitempty"`
	LogPath        string                    `json:"logPath"`
	Status         string                    `json:"status"`
	PID            int                       `json:"pid"`
	LastStartedAt  string                    `json:"lastStartedAt,omitempty"`
	LastExitedAt   string                    `json:"lastExitedAt,omitempty"`
	LastExitError  string                    `json:"lastExitError,omitempty"`
	DesiredRunning bool                      `json:"desiredRunning"`
	PortForward    *ServicePortForwardStatus `json:"portForward,omitempty"`
}

type serviceProcess struct {
	def                 ServiceDefinition
	cmd                 *exec.Cmd
	pid                 int
	status              string
	logPath             string
	lastStartedAt       string
	lastExitedAt        string
	lastExitError       string
	desired             bool
	stopRequested       bool
	ownedForward        bool
	runStartedAt        time.Time
	consecutiveFailures int
	nextRestartAt       time.Time
}

type Manager struct {
	mu            sync.Mutex
	definitions   []ServiceDefinition
	processes     map[string]*serviceProcess
	healthStop    chan struct{}
	healthStarted bool
}

var (
	defaultManager     = NewManager()
	servicesConfigPath = filepath.Join(config.DataDir, "services.json")
)

func NewManager() *Manager {
	m := &Manager{
		processes: make(map[string]*serviceProcess),
	}
	m.loadDefinitionsLocked()
	return m
}

func GetDefaultManager() *Manager {
	return defaultManager
}

func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/services", handleServices)
	mux.HandleFunc("/api/services/start", handleStartService)
	mux.HandleFunc("/api/services/stop", handleStopService)
	mux.HandleFunc("/api/services/restart", handleRestartService)
}

func StartHealthCheck() {
	defaultManager.StartHealthCheck()
}

func AutoStartConfiguredServices() {
	defaultManager.AutoStartConfiguredServices()
}

func Shutdown() {
	defaultManager.Shutdown()
}

func (m *Manager) loadDefinitionsLocked() {
	data, err := os.ReadFile(servicesConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			m.definitions = []ServiceDefinition{}
		}
		return
	}

	var defs []ServiceDefinition
	if err := json.Unmarshal(data, &defs); err != nil {
		fmt.Printf("[services] failed to parse services config: %v\n", err)
		return
	}

	for i := range defs {
		defs[i].ProjectDir = normalizeProjectDir(defs[i].ProjectDir)
		defs[i].WorkingDir = normalizeWorkingDir(defs[i].WorkingDir)
		defs[i].ExtraEnv = normalizeExtraEnv(defs[i].ExtraEnv)
	}
	m.definitions = defs
}

func (m *Manager) saveDefinitionsLocked() error {
	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m.definitions, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(servicesConfigPath, data, 0644)
}

func (m *Manager) StartHealthCheck() {
	m.mu.Lock()
	if m.healthStarted {
		m.mu.Unlock()
		return
	}
	m.healthStarted = true
	m.healthStop = make(chan struct{})
	stopCh := m.healthStop
	m.mu.Unlock()

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				m.reconcileProcesses()
			}
		}
	}()
}

func (m *Manager) AutoStartConfiguredServices() {
	defs := m.getDefinitions("")
	for _, def := range defs {
		if _, err := m.Start(def.ID); err != nil {
			fmt.Printf("[services] failed to autostart %s: %v\n", def.Name, err)
		}
	}
}

func (m *Manager) Shutdown() {
	m.mu.Lock()
	var ids []string
	if m.healthStarted && m.healthStop != nil {
		close(m.healthStop)
		m.healthStarted = false
		m.healthStop = nil
	}
	for id := range m.processes {
		ids = append(ids, id)
	}
	m.mu.Unlock()

	for _, id := range ids {
		_ = m.stop(id, true, true)
	}
}

func (m *Manager) List(projectDir string) []ServiceStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	projectDir = normalizeProjectDir(projectDir)
	defs := m.filteredDefinitionsLocked(projectDir)
	portMap := make(map[int]portforward.PortForward)
	for _, pf := range portforward.GetDefaultManager().List() {
		portMap[pf.LocalPort] = pf
	}

	result := make([]ServiceStatus, 0, len(defs))
	for _, def := range defs {
		proc := m.processes[def.ID]
		serviceEnv := buildServiceEnv(def)
		logPath := serviceLogPath(def.ID)
		status := StatusStopped
		pid := 0
		lastStartedAt := ""
		lastExitedAt := ""
		lastExitError := ""
		desired := true
		ownedForward := false
		if proc != nil {
			status = proc.status
			pid = proc.pid
			lastStartedAt = proc.lastStartedAt
			lastExitedAt = proc.lastExitedAt
			lastExitError = proc.lastExitError
			desired = proc.desired
			ownedForward = proc.ownedForward
			if proc.logPath != "" {
				logPath = proc.logPath
			}
			if pid > 0 && !processAlive(pid) {
				pid = 0
				if status == StatusRunning {
					status = StatusStopped
				}
			}
		}

		var pfStatus *ServicePortForwardStatus
		if def.PortForward != nil && def.PortForward.Port > 0 {
			pfStatus = &ServicePortForwardStatus{
				Port:       def.PortForward.Port,
				Label:      def.PortForward.Label,
				Provider:   normalizeProvider(def.PortForward.Provider),
				BaseDomain: def.PortForward.BaseDomain,
				Subdomain:  def.PortForward.Subdomain,
			}
			if active, ok := portMap[def.PortForward.Port]; ok {
				pfStatus.PublicURL = active.PublicURL
				pfStatus.Status = active.Status
				pfStatus.Error = active.Error
				pfStatus.Active = active.Status == portforward.StatusActive || ownedForward
				if active.Provider != "" {
					pfStatus.Provider = active.Provider
				}
				if active.Label != "" && pfStatus.Label == "" {
					pfStatus.Label = active.Label
				}
			}
		}

		result = append(result, ServiceStatus{
			ID:             def.ID,
			Name:           def.Name,
			Command:        def.Command,
			ProjectDir:     def.ProjectDir,
			WorkingDir:     def.WorkingDir,
			ExtraEnv:       cloneStringMap(def.ExtraEnv),
			EffectivePath:  lookupEnvValue(serviceEnv, "PATH"),
			LogPath:        logPath,
			Status:         status,
			PID:            pid,
			LastStartedAt:  lastStartedAt,
			LastExitedAt:   lastExitedAt,
			LastExitError:  lastExitError,
			DesiredRunning: desired,
			PortForward:    pfStatus,
		})
	}
	return result
}

func (m *Manager) CreateOrUpdate(def ServiceDefinition) (*ServiceStatus, error) {
	def.ProjectDir = normalizeProjectDir(def.ProjectDir)
	def.WorkingDir = normalizeWorkingDir(def.WorkingDir)
	def.ExtraEnv = normalizeExtraEnv(def.ExtraEnv)
	if err := validateDefinition(def); err != nil {
		return nil, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	var existing *ServiceDefinition
	var shouldRestart bool

	m.mu.Lock()
	if def.ID == "" {
		def.ID = fmt.Sprintf("svc-%d", time.Now().UnixNano())
		def.CreatedAt = now
		def.UpdatedAt = now
		m.definitions = append(m.definitions, def)
	} else {
		for i := range m.definitions {
			if m.definitions[i].ID != def.ID {
				continue
			}
			existingCopy := m.definitions[i]
			existing = &existingCopy
			def.CreatedAt = m.definitions[i].CreatedAt
			def.UpdatedAt = now
			m.definitions[i] = def
			break
		}
		if existing == nil {
			def.CreatedAt = now
			def.UpdatedAt = now
			m.definitions = append(m.definitions, def)
		}
	}
	if err := m.saveDefinitionsLocked(); err != nil {
		m.mu.Unlock()
		return nil, err
	}

	proc := m.processes[def.ID]
	if proc != nil {
		proc.def = def
		shouldRestart = proc.desired && (existing == nil || definitionChanged(*existing, def))
	}
	m.mu.Unlock()

	if shouldRestart {
		if err := m.Restart(def.ID); err != nil {
			return nil, err
		}
	}

	list := m.List(def.ProjectDir)
	for _, item := range list {
		if item.ID == def.ID {
			copy := item
			return &copy, nil
		}
	}
	return nil, fmt.Errorf("service %s not found after save", def.ID)
}

func (m *Manager) Delete(id string) error {
	if err := m.stop(id, true, true); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	filtered := make([]ServiceDefinition, 0, len(m.definitions))
	found := false
	for _, def := range m.definitions {
		if def.ID == id {
			found = true
			continue
		}
		filtered = append(filtered, def)
	}
	if !found {
		return fmt.Errorf("service not found")
	}
	m.definitions = filtered
	return m.saveDefinitionsLocked()
}

func (m *Manager) Start(id string) (*ServiceStatus, error) {
	if err := m.start(id, true); err != nil {
		return nil, err
	}

	m.mu.Lock()
	projectDir := ""
	for _, def := range m.definitions {
		if def.ID == id {
			projectDir = def.ProjectDir
			break
		}
	}
	m.mu.Unlock()

	list := m.List(projectDir)
	for _, item := range list {
		if item.ID == id {
			copy := item
			return &copy, nil
		}
	}
	return nil, fmt.Errorf("service not found")
}

func (m *Manager) Restart(id string) error {
	if err := m.stop(id, false, true); err != nil {
		return err
	}
	return m.start(id, true)
}

func (m *Manager) start(id string, force bool) error {
	m.mu.Lock()
	def, ok := m.findDefinitionLocked(id)
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("service not found")
	}

	proc := m.processes[id]
	if proc == nil {
		proc = &serviceProcess{
			def:           def,
			status:        StatusStopped,
			logPath:       serviceLogPath(id),
			desired:       true,
			stopRequested: false,
		}
		m.processes[id] = proc
	} else {
		proc.def = def
		proc.desired = true
		proc.stopRequested = false
		if proc.logPath == "" {
			proc.logPath = serviceLogPath(id)
		}
	}

	if proc.pid > 0 && processAlive(proc.pid) {
		proc.status = StatusRunning
		m.mu.Unlock()
		return nil
	}

	if !force && !proc.nextRestartAt.IsZero() && time.Now().Before(proc.nextRestartAt) {
		proc.status = StatusError
		m.mu.Unlock()
		return nil
	}

	proc.status = StatusStarting
	proc.lastExitError = ""
	logPath := proc.logPath
	workingDir := def.WorkingDir
	m.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		m.recordStartFailure(id, err.Error())
		return err
	}

	if err := killConfiguredPortOwner(def.PortForward); err != nil {
		m.recordStartFailure(id, err.Error())
		return err
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		m.recordStartFailure(id, err.Error())
		return err
	}

	startMarker := fmt.Sprintf("\n[%s] starting service %s\n", time.Now().Format(time.RFC3339), def.Name)
	_, _ = logFile.WriteString(startMarker)

	serviceEnv := buildServiceEnv(def)
	shellCommand := def.Command
	if pathVal := lookupEnvValue(serviceEnv, "PATH"); pathVal != "" {
		// Keep login-shell behavior, but restore the resolved PATH after
		// profile scripts run. Verified locally:
		//   bash -c  'echo $PATH'  -> keeps the injected tool_resolve PATH entries
		//   bash -lc 'echo $PATH'  -> login startup files reset PATH and drop them
		// So we prepend an explicit export for PATH before running the command.
		shellCommand = fmt.Sprintf("export PATH=%s; %s", shellQuote(pathVal), def.Command)
	}
	cmd := exec.Command("bash", "-lc", shellCommand)
	cmd.Dir = workingDir
	// Match terminal/tool execution behavior so managed services can find
	// binaries installed in tool_resolve's extra PATH entries.
	cmd.Env = serviceEnv
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		_, _ = logFile.WriteString(fmt.Sprintf("[%s] failed to start: %v\n", time.Now().Format(time.RFC3339), err))
		logFile.Close()
		m.recordStartFailure(id, err.Error())
		return err
	}

	pid := cmd.Process.Pid
	startedAt := time.Now()

	m.mu.Lock()
	proc = m.processes[id]
	if proc == nil {
		proc = &serviceProcess{}
		m.processes[id] = proc
	}
	proc.def = def
	proc.cmd = cmd
	proc.pid = pid
	proc.status = StatusRunning
	proc.lastStartedAt = startedAt.UTC().Format(time.RFC3339)
	proc.desired = true
	proc.stopRequested = false
	proc.logPath = logPath
	proc.runStartedAt = startedAt
	proc.nextRestartAt = time.Time{}
	proc.lastExitError = ""
	m.mu.Unlock()

	if err := m.ensurePortForward(id, def); err != nil {
		_, _ = logFile.WriteString(fmt.Sprintf("[%s] failed to ensure port forwarding: %v\n", time.Now().Format(time.RFC3339), err))
		m.setRuntimeError(id, err.Error())
	}

	go m.waitForExit(id, cmd, logFile)
	return nil
}

func (m *Manager) waitForExit(id string, cmd *exec.Cmd, logFile *os.File) {
	err := cmd.Wait()
	exitAt := time.Now()
	exitTime := exitAt.UTC().Format(time.RFC3339)
	if err != nil {
		_, _ = logFile.WriteString(fmt.Sprintf("[%s] service exited with error: %v\n", time.Now().Format(time.RFC3339), err))
	} else {
		_, _ = logFile.WriteString(fmt.Sprintf("[%s] service exited\n", time.Now().Format(time.RFC3339)))
	}
	_ = logFile.Close()

	m.mu.Lock()
	proc := m.processes[id]
	if proc != nil && proc.cmd == cmd {
		runDuration := time.Duration(0)
		if !proc.runStartedAt.IsZero() {
			runDuration = exitAt.Sub(proc.runStartedAt)
		}
		proc.pid = 0
		proc.cmd = nil
		proc.runStartedAt = time.Time{}
		proc.lastExitedAt = exitTime
		if proc.stopRequested {
			proc.status = StatusStopped
			proc.lastExitError = ""
			proc.consecutiveFailures = 0
			proc.nextRestartAt = time.Time{}
		} else {
			if runDuration >= stableRunResetWindow {
				proc.consecutiveFailures = 0
			}
			proc.consecutiveFailures++
			proc.nextRestartAt = computeBackoffTime(proc.consecutiveFailures)
			if err != nil {
				proc.lastExitError = err.Error()
			} else {
				proc.lastExitError = fmt.Sprintf("service exited unexpectedly; retrying in %s", proc.nextRestartAt.Sub(exitAt).Round(time.Second))
			}
			proc.status = StatusError
		}
		proc.stopRequested = false
	}
	m.mu.Unlock()
}

func (m *Manager) stop(id string, removeForward bool, wait bool) error {
	m.mu.Lock()
	proc, ok := m.processes[id]
	if !ok {
		_, exists := m.findDefinitionLocked(id)
		m.mu.Unlock()
		if !exists {
			return fmt.Errorf("service not found")
		}
		return nil
	}

	proc.desired = false
	proc.stopRequested = true
	proc.status = StatusStopped
	pid := proc.pid
	ownedForward := proc.ownedForward
	port := 0
	if proc.def.PortForward != nil {
		port = proc.def.PortForward.Port
	}
	if pid == 0 {
		proc.cmd = nil
		proc.ownedForward = false
		proc.consecutiveFailures = 0
		proc.nextRestartAt = time.Time{}
		proc.runStartedAt = time.Time{}
	}
	m.mu.Unlock()

	if removeForward && ownedForward && port > 0 {
		_ = portforward.GetDefaultManager().Remove(port)
		m.mu.Lock()
		if current := m.processes[id]; current != nil {
			current.ownedForward = false
		}
		m.mu.Unlock()
	}

	if pid == 0 {
		return nil
	}

	if err := stopProcessGroup(pid); err != nil {
		return err
	}

	if wait {
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if !processAlive(pid) {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	return nil
}

func (m *Manager) ensurePortForward(id string, def ServiceDefinition) error {
	if def.PortForward == nil || def.PortForward.Port <= 0 {
		return nil
	}

	pf := def.PortForward
	manager := portforward.GetDefaultManager()
	desiredProvider := normalizeProvider(pf.Provider)
	desiredLabel := resolveForwardLabel(pf)

	for _, existing := range manager.List() {
		if existing.LocalPort != pf.Port {
			continue
		}
		if existing.Provider == desiredProvider && existing.Label == desiredLabel {
			m.mu.Lock()
			if proc := m.processes[id]; proc != nil {
				proc.ownedForward = false
			}
			m.mu.Unlock()
			return nil
		}
		return nil
	}

	if _, err := manager.Add(pf.Port, desiredLabel, desiredProvider); err != nil {
		return err
	}

	m.mu.Lock()
	if proc := m.processes[id]; proc != nil {
		proc.ownedForward = true
	}
	m.mu.Unlock()
	return nil
}

func (m *Manager) reconcileProcesses() {
	var restartIDs []string

	m.mu.Lock()
	now := time.Now()
	for id, proc := range m.processes {
		if proc == nil || !proc.desired {
			continue
		}
		if proc.pid > 0 && processAlive(proc.pid) {
			continue
		}
		if proc.status == StatusStarting {
			continue
		}
		if !proc.nextRestartAt.IsZero() && now.Before(proc.nextRestartAt) {
			continue
		}
		restartIDs = append(restartIDs, id)
	}
	m.mu.Unlock()

	for _, id := range restartIDs {
		if err := m.start(id, false); err != nil {
			fmt.Printf("[services] healthcheck restart failed for %s: %v\n", id, err)
		}
	}
}

func (m *Manager) setError(id string, err error) {
	m.recordStartFailure(id, err.Error())
}

func (m *Manager) setRuntimeError(id string, msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if proc := m.processes[id]; proc != nil {
		proc.pid = 0
		proc.cmd = nil
		proc.runStartedAt = time.Time{}
		proc.status = StatusError
		proc.lastExitError = msg
		proc.lastExitedAt = time.Now().UTC().Format(time.RFC3339)
	}
}

func (m *Manager) recordStartFailure(id string, msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if proc := m.processes[id]; proc != nil {
		now := time.Now()
		proc.pid = 0
		proc.cmd = nil
		proc.runStartedAt = time.Time{}
		proc.status = StatusError
		proc.lastExitError = msg
		proc.lastExitedAt = now.UTC().Format(time.RFC3339)
		proc.consecutiveFailures++
		proc.nextRestartAt = now.Add(computeBackoffDelay(proc.consecutiveFailures))
	}
}

func computeBackoffTime(failures int) time.Time {
	return time.Now().Add(computeBackoffDelay(failures))
}

func computeBackoffDelay(failures int) time.Duration {
	if failures <= 0 {
		return 0
	}
	delay := restartBackoffBase
	for i := 1; i < failures; i++ {
		if delay >= restartBackoffMax/2 {
			return restartBackoffMax
		}
		delay *= 2
	}
	if delay > restartBackoffMax {
		return restartBackoffMax
	}
	return delay
}

func (m *Manager) getDefinitions(projectDir string) []ServiceDefinition {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]ServiceDefinition(nil), m.filteredDefinitionsLocked(normalizeProjectDir(projectDir))...)
}

func (m *Manager) filteredDefinitionsLocked(projectDir string) []ServiceDefinition {
	if projectDir == "" {
		return append([]ServiceDefinition(nil), m.definitions...)
	}
	filtered := make([]ServiceDefinition, 0, len(m.definitions))
	for _, def := range m.definitions {
		if normalizeProjectDir(def.ProjectDir) == projectDir {
			filtered = append(filtered, def)
		}
	}
	return filtered
}

func (m *Manager) findDefinitionLocked(id string) (ServiceDefinition, bool) {
	for _, def := range m.definitions {
		if def.ID == id {
			return def, true
		}
	}
	return ServiceDefinition{}, false
}

func normalizeProvider(provider string) string {
	if provider == "" {
		return portforward.ProviderLocaltunnel
	}
	return provider
}

func validateDefinition(def ServiceDefinition) error {
	if strings.TrimSpace(def.Name) == "" {
		return fmt.Errorf("service name is required")
	}
	if strings.TrimSpace(def.Command) == "" {
		return fmt.Errorf("service command is required")
	}
	for key := range def.ExtraEnv {
		if !isValidEnvKey(key) {
			return fmt.Errorf("invalid environment variable name: %s", key)
		}
	}
	if def.PortForward != nil {
		if def.PortForward.Port <= 0 || def.PortForward.Port > 65535 {
			return fmt.Errorf("service port must be between 1 and 65535")
		}
	}
	return nil
}

func definitionChanged(oldDef ServiceDefinition, newDef ServiceDefinition) bool {
	if oldDef.Name != newDef.Name || oldDef.Command != newDef.Command || oldDef.ProjectDir != newDef.ProjectDir || oldDef.WorkingDir != newDef.WorkingDir {
		return true
	}
	if !stringMapEqual(oldDef.ExtraEnv, newDef.ExtraEnv) {
		return true
	}

	oldPF := oldDef.PortForward
	newPF := newDef.PortForward
	if (oldPF == nil) != (newPF == nil) {
		return true
	}
	if oldPF == nil {
		return false
	}
	return oldPF.Port != newPF.Port ||
		oldPF.Label != newPF.Label ||
		oldPF.Provider != newPF.Provider ||
		oldPF.BaseDomain != newPF.BaseDomain ||
		oldPF.Subdomain != newPF.Subdomain
}

func resolveForwardLabel(pf *ServicePortForward) string {
	if pf == nil {
		return ""
	}
	if pf.Label != "" {
		return pf.Label
	}
	if pf.Subdomain != "" {
		if pf.BaseDomain != "" {
			return fmt.Sprintf("%s.%s", pf.Subdomain, pf.BaseDomain)
		}
		ownedDomains := cloudflare.GetOwnedDomains()
		if len(ownedDomains) > 0 {
			return fmt.Sprintf("%s.%s", pf.Subdomain, ownedDomains[0])
		}
	}
	return fmt.Sprintf("Port %d", pf.Port)
}

func killConfiguredPortOwner(pf *ServicePortForward) error {
	if pf == nil || pf.Port <= 0 {
		return nil
	}
	pids, err := findListeningPIDs(pf.Port)
	if err != nil {
		return err
	}
	for _, pid := range pids {
		if pid == os.Getpid() {
			return fmt.Errorf("refusing to kill current server process on port %d", pf.Port)
		}
		if err := stopProcessGroup(pid); err != nil {
			return fmt.Errorf("failed to free port %d: %w", pf.Port, err)
		}
	}
	return nil
}

func findListeningPIDs(port int) ([]int, error) {
	cmd := exec.Command("lsof", "-tiTCP:"+strconv.Itoa(port), "-sTCP:LISTEN", "-n", "-P")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) == 0 {
			return nil, nil
		}
		if execErr, ok := err.(*exec.Error); ok && execErr.Err == exec.ErrNotFound {
			return nil, fmt.Errorf("lsof not installed: required for service port management")
		}
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var pids []int
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pid, convErr := strconv.Atoi(line)
		if convErr == nil && pid > 0 {
			pids = append(pids, pid)
		}
	}
	return pids, nil
}

func stopProcessGroup(pid int) error {
	if pid <= 0 {
		return nil
	}

	_ = syscall.Kill(-pid, syscall.SIGTERM)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	_ = syscall.Kill(-pid, syscall.SIGKILL)
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("process %d did not exit", pid)
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

func serviceLogPath(id string) string {
	return filepath.Join(config.DataDir, "services", id+".log")
}

func normalizeProjectDir(projectDir string) string {
	projectDir = strings.TrimSpace(projectDir)
	if projectDir == "" {
		projectDir = config.GetServerProjectDir()
		if projectDir == "" {
			if cwd, err := os.Getwd(); err == nil {
				projectDir = cwd
			}
		}
	}
	if projectDir == "" {
		return ""
	}
	abs, err := filepath.Abs(projectDir)
	if err != nil {
		return projectDir
	}
	return abs
}

func normalizeWorkingDir(workingDir string) string {
	workingDir = strings.TrimSpace(workingDir)
	if workingDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			workingDir = strings.TrimSpace(home)
		}
		if workingDir == "" {
			if cwd, err := os.Getwd(); err == nil {
				workingDir = cwd
			}
		}
	}
	if workingDir == "" {
		return ""
	}
	abs, err := filepath.Abs(workingDir)
	if err != nil {
		return workingDir
	}
	return abs
}

func normalizeExtraEnv(extraEnv map[string]string) map[string]string {
	if len(extraEnv) == 0 {
		return nil
	}
	normalized := make(map[string]string, len(extraEnv))
	for key, value := range extraEnv {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		normalized[key] = value
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func buildServiceEnv(def ServiceDefinition) []string {
	env := tool_resolve.AppendExtraPaths(os.Environ())
	if len(def.ExtraEnv) == 0 {
		return env
	}
	envMap := make(map[string]string, len(env)+len(def.ExtraEnv))
	for _, item := range env {
		if idx := strings.IndexByte(item, '='); idx >= 0 {
			envMap[item[:idx]] = item[idx+1:]
		}
	}
	for key, value := range def.ExtraEnv {
		envMap[key] = value
	}
	merged := make([]string, 0, len(envMap))
	for key, value := range envMap {
		merged = append(merged, fmt.Sprintf("%s=%s", key, value))
	}
	return merged
}

func lookupEnvValue(env []string, key string) string {
	prefix := key + "="
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			return strings.TrimPrefix(item, prefix)
		}
	}
	return ""
}

func isValidEnvKey(key string) bool {
	if key == "" {
		return false
	}
	for i, ch := range key {
		if i == 0 {
			if ch != '_' && !(ch >= 'A' && ch <= 'Z') && !(ch >= 'a' && ch <= 'z') {
				return false
			}
			continue
		}
		if ch != '_' && !(ch >= 'A' && ch <= 'Z') && !(ch >= 'a' && ch <= 'z') && !(ch >= '0' && ch <= '9') {
			return false
		}
	}
	return true
}

func stringMapEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if b[key] != value {
			return false
		}
	}
	return true
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
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

func handleServices(w http.ResponseWriter, r *http.Request) {
	manager := GetDefaultManager()

	switch r.Method {
	case http.MethodGet:
		projectDir := r.URL.Query().Get("project_dir")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(manager.List(projectDir))

	case http.MethodPost, http.MethodPut:
		var req ServiceDefinition
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		saved, err := manager.CreateOrUpdate(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
		}
		_ = json.NewEncoder(w).Encode(saved)

	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id is required", http.StatusBadRequest)
			return
		}
		if err := manager.Delete(id); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleStartService(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	status, err := GetDefaultManager().Start(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

func handleStopService(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	if err := GetDefaultManager().stop(id, true, true); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleRestartService(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	if err := GetDefaultManager().Restart(id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
