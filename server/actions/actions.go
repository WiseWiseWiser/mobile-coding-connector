package actions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
	"github.com/xhd2015/lifelog-private/ai-critic/server/sse"
	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_resolve"
)

// Action represents a user-defined custom action
type Action struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Icon   string `json:"icon"`
	Script string `json:"script"`
}

// ActionStatus represents the running status of an action
type ActionStatus struct {
	ActionID   string    `json:"action_id"`
	Running    bool      `json:"running"`
	StartedAt  time.Time `json:"started_at,omitempty"`
	FinishedAt time.Time `json:"finished_at,omitempty"`
	Logs       []string  `json:"logs"`
	ExitCode   int       `json:"exit_code,omitempty"`
	PID        int       `json:"pid,omitempty"`
}

// ActionRunRequest represents a request to run an action
type ActionRunRequest struct {
	ProjectDir string `json:"project_dir"`
	Script     string `json:"script"`
}

// ActionStateFile represents the persisted state of actions
type ActionStateFile struct {
	Statuses map[string]ActionStatus `json:"statuses"`
}

var (
	baseDir         = config.ProjectsDir
	mu              sync.RWMutex
	actionStatuses  = make(map[string]*ActionStatus)
	actionProcesses = make(map[string]*exec.Cmd)
)

// getActionsDir returns the directory for storing actions.json
type ActionsDir interface {
	ActionsDir(projectName string) string
}

type fileSystemActionsDir struct{}

func (fs fileSystemActionsDir) ActionsDir(projectName string) string {
	return filepath.Join(baseDir, projectName)
}

var actionsDirImpl ActionsDir = fileSystemActionsDir{}

func projectActionsDir(projectName string) string {
	return actionsDirImpl.ActionsDir(projectName)
}

// getActionsFile returns the path to the actions.json file
func getActionsFile(projectName string) string {
	return filepath.Join(projectActionsDir(projectName), "actions.json")
}

// ensureProjectDir ensures the project directory exists
func ensureProjectDir(projectName string) error {
	return os.MkdirAll(projectActionsDir(projectName), 0755)
}

// getStateFile returns the path to the action state file
func getStateFile(projectName string) string {
	return filepath.Join(projectActionsDir(projectName), "actions_state.json")
}

// loadState loads the persisted action state
func loadState(projectName string) (map[string]ActionStatus, error) {
	stateFile := getStateFile(projectName)
	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]ActionStatus), nil
		}
		return nil, err
	}

	var state ActionStateFile
	if err := json.Unmarshal(data, &state); err != nil {
		return make(map[string]ActionStatus), nil
	}

	// Clear stale running statuses on load (processes don't survive restart)
	for id, status := range state.Statuses {
		if status.Running {
			status.Running = false
			status.FinishedAt = time.Now()
			state.Statuses[id] = status
		}
	}

	return state.Statuses, nil
}

// saveState persists the action state to disk
func saveState(projectName string, statuses map[string]ActionStatus) error {
	if err := ensureProjectDir(projectName); err != nil {
		return err
	}

	stateFile := getStateFile(projectName)
	state := ActionStateFile{Statuses: statuses}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(stateFile, data, 0644)
}

// InitState loads persisted state for a project
func InitState(projectName string) {
	mu.Lock()
	defer mu.Unlock()

	statuses, err := loadState(projectName)
	if err == nil && statuses != nil {
		for id, status := range statuses {
			actionStatuses[id] = &status
		}
	}
}

// GetActionStatus returns the status of a specific action
func GetActionStatus(actionID string) *ActionStatus {
	mu.RLock()
	defer mu.RUnlock()
	return actionStatuses[actionID]
}

// GetAllActionStatuses returns all action statuses for a project
func GetAllActionStatuses(projectName string) map[string]ActionStatus {
	mu.RLock()
	defer mu.RUnlock()

	statuses, err := loadState(projectName)
	if err != nil {
		return make(map[string]ActionStatus)
	}

	// Merge with in-memory statuses
	result := make(map[string]ActionStatus)
	for id, status := range statuses {
		if memStatus, ok := actionStatuses[id]; ok && memStatus.Running {
			result[id] = *memStatus
		} else {
			result[id] = status
		}
	}
	return result
}

// LoadActions loads all actions for a project
func LoadActions(projectName string) ([]Action, error) {
	mu.RLock()
	defer mu.RUnlock()

	actionsFile := getActionsFile(projectName)
	data, err := os.ReadFile(actionsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []Action{}, nil
		}
		return nil, err
	}

	var actions []Action
	if err := json.Unmarshal(data, &actions); err != nil {
		return nil, err
	}

	return actions, nil
}

// SaveActions saves all actions for a project
func SaveActions(projectName string, actions []Action) error {
	mu.Lock()
	defer mu.Unlock()

	if err := ensureProjectDir(projectName); err != nil {
		return err
	}

	actionsFile := getActionsFile(projectName)
	data, err := json.MarshalIndent(actions, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(actionsFile, data, 0644)
}

// AddAction adds a new action to a project
func AddAction(projectName string, action Action) error {
	actions, err := LoadActions(projectName)
	if err != nil {
		return err
	}

	// Generate ID if not provided
	if action.ID == "" {
		action.ID = generateActionID(actions)
	}

	actions = append(actions, action)
	return SaveActions(projectName, actions)
}

// UpdateAction updates an existing action
func UpdateAction(projectName string, action Action) error {
	actions, err := LoadActions(projectName)
	if err != nil {
		return err
	}

	found := false
	for i, a := range actions {
		if a.ID == action.ID {
			actions[i] = action
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("action %s not found", action.ID)
	}

	return SaveActions(projectName, actions)
}

// DeleteAction deletes an action by ID
func DeleteAction(projectName string, actionID string) error {
	actions, err := LoadActions(projectName)
	if err != nil {
		return err
	}

	found := false
	for i, a := range actions {
		if a.ID == actionID {
			actions = append(actions[:i], actions[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("action %s not found", actionID)
	}

	return SaveActions(projectName, actions)
}

// generateActionID generates a unique ID for a new action
func generateActionID(actions []Action) string {
	maxID := 0
	for _, a := range actions {
		if strings.HasPrefix(a.ID, "action_") {
			var id int
			fmt.Sscanf(a.ID, "action_%d", &id)
			if id > maxID {
				maxID = id
			}
		}
	}
	return fmt.Sprintf("action_%d", maxID+1)
}

// RunAction executes a script in the project directory with SSE streaming
func RunAction(projectDir string, script string, w http.ResponseWriter) error {
	return RunActionWithID("", projectDir, script, w)
}

// RunActionWithID executes a script with specific action ID for tracking
func RunActionWithID(actionID string, projectDir string, script string, w http.ResponseWriter) error {
	sw := sse.NewWriter(w)
	if sw == nil {
		return fmt.Errorf("streaming not supported")
	}

	// Set up status tracking if actionID provided
	var status *ActionStatus
	if actionID != "" {
		status = &ActionStatus{
			ActionID:  actionID,
			Running:   true,
			StartedAt: time.Now(),
			Logs:      []string{},
		}
		mu.Lock()
		actionStatuses[actionID] = status
		mu.Unlock()
	}

	log := func(msg string) {
		sw.SendLog(msg)
		if status != nil {
			mu.Lock()
			status.Logs = append(status.Logs, msg)
			mu.Unlock()
		}
	}

	log(fmt.Sprintf("Running action in %s...", projectDir))

	// Create command using shell
	var cmd *exec.Cmd
	if strings.Contains(script, "\n") || strings.Contains(script, ";") || strings.Contains(script, "&&") {
		// Multi-line or complex command - use shell
		cmd = exec.Command("bash", "-c", script)
	} else {
		// Simple command - parse and execute directly
		parts := strings.Fields(script)
		if len(parts) == 0 {
			log("Empty script")
			sw.SendDone(map[string]string{"success": "false", "message": "Empty script"})
			clearStatus(actionID, status, -1)
			return nil
		}
		cmd = exec.Command(parts[0], parts[1:]...)
	}

	// Store cmd for stopping
	if actionID != "" && cmd != nil {
		mu.Lock()
		actionProcesses[actionID] = cmd
		mu.Unlock()
	}

	cmd.Dir = projectDir

	// Set up environment with PATH additions
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	cmd.Env = tool_resolve.AppendExtraPaths(cmd.Env)

	// Custom output handler to capture logs
	cmd.Stdout = &logWriter{log: log}
	cmd.Stderr = &logWriter{log: log}

	// Run the command
	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
		log(fmt.Sprintf("Action failed: %v", err))
		sw.SendDone(map[string]string{"success": "false", "message": err.Error()})
	} else {
		log("Action completed successfully")
		sw.SendDone(map[string]string{"success": "true", "message": "Action completed successfully"})
	}

	clearStatus(actionID, status, exitCode)
	return nil
}

// logWriter writes to both stdout/stderr and captures logs
type logWriter struct {
	log func(string)
}

func (lw *logWriter) Write(p []byte) (n int, err error) {
	lw.log(strings.TrimSpace(string(p)))
	return len(p), nil
}

// clearStatus clears the running status after completion
func clearStatus(actionID string, status *ActionStatus, exitCode int) {
	if status == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()

	status.Running = false
	status.FinishedAt = time.Now()
	status.ExitCode = exitCode
	if cmd, ok := actionProcesses[actionID]; ok && cmd != nil {
		actionProcesses[actionID] = nil
	}
	delete(actionProcesses, actionID)

	// Save to disk
	projectName := filepath.Base(filepath.Dir(getActionsFile("")))
	statuses := make(map[string]ActionStatus)
	statuses[actionID] = *status
	saveState(projectName, statuses)
}

// StopAction stops a running action by ID
func StopAction(actionID string) error {
	mu.Lock()
	defer mu.Unlock()

	cmd, ok := actionProcesses[actionID]
	if !ok || cmd == nil || cmd.Process == nil {
		return fmt.Errorf("action %s is not running", actionID)
	}

	// Try to kill the process
	err := cmd.Process.Kill()
	if err != nil {
		return fmt.Errorf("failed to stop action: %v", err)
	}

	// Update status
	if status, ok := actionStatuses[actionID]; ok {
		status.Running = false
		status.FinishedAt = time.Now()
		status.ExitCode = -1
		status.Logs = append(status.Logs, "Action stopped by user")

		// Save to disk
		projectName := filepath.Base(filepath.Dir(getActionsFile("")))
		statuses := make(map[string]ActionStatus)
		statuses[actionID] = *status
		saveState(projectName, statuses)
	}

	actionProcesses[actionID] = nil
	delete(actionProcesses, actionID)

	return nil
}

// RegisterAPI registers the actions API endpoints
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/actions", handleActions)
	mux.HandleFunc("/api/actions/", handleActionByID)
	mux.HandleFunc("/api/actions/run", handleRunAction)
	mux.HandleFunc("/api/actions/status", handleActionStatus)
	mux.HandleFunc("/api/actions/stop", handleActionStop)
}

func handleActions(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	if project == "" {
		respondErr(w, http.StatusBadRequest, "project is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		actions, err := LoadActions(project)
		if err != nil {
			respondErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, actions)

	case http.MethodPost:
		var action Action
		if err := json.NewDecoder(r.Body).Decode(&action); err != nil {
			respondErr(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if action.Name == "" {
			respondErr(w, http.StatusBadRequest, "name is required")
			return
		}

		if action.Script == "" {
			respondErr(w, http.StatusBadRequest, "script is required")
			return
		}

		if err := AddAction(project, action); err != nil {
			respondErr(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, action)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleActionByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/actions/")

	project := r.URL.Query().Get("project")
	if project == "" {
		respondErr(w, http.StatusBadRequest, "project is required")
		return
	}

	actionID := path
	if actionID == "" {
		respondErr(w, http.StatusBadRequest, "action ID is required")
		return
	}

	switch r.Method {
	case http.MethodPut:
		var action Action
		if err := json.NewDecoder(r.Body).Decode(&action); err != nil {
			respondErr(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if action.ID != actionID {
			respondErr(w, http.StatusBadRequest, "action ID mismatch")
			return
		}

		if err := UpdateAction(project, action); err != nil {
			respondErr(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, action)

	case http.MethodDelete:
		if err := DeleteAction(project, actionID); err != nil {
			respondErr(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleRunAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ActionRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ProjectDir == "" {
		respondErr(w, http.StatusBadRequest, "project_dir is required")
		return
	}

	if req.Script == "" {
		respondErr(w, http.StatusBadRequest, "script is required")
		return
	}

	// Run the action with SSE streaming
	if err := RunAction(req.ProjectDir, req.Script, w); err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
	}
}

func handleActionStatus(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	if project == "" {
		respondErr(w, http.StatusBadRequest, "project is required")
		return
	}

	actionID := r.URL.Query().Get("action_id")

	if actionID != "" {
		// Get specific action status
		status := GetActionStatus(actionID)
		if status == nil {
			// Load from disk
			statuses, _ := loadState(project)
			if s, ok := statuses[actionID]; ok {
				respondJSON(w, http.StatusOK, s)
				return
			}
			respondJSON(w, http.StatusOK, ActionStatus{ActionID: actionID})
			return
		}
		respondJSON(w, http.StatusOK, status)
		return
	}

	// Get all action statuses for project
	statuses := GetAllActionStatuses(project)
	respondJSON(w, http.StatusOK, statuses)
}

func handleActionStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	actionID := r.URL.Query().Get("action_id")
	if actionID == "" {
		respondErr(w, http.StatusBadRequest, "action_id is required")
		return
	}

	err := StopAction(actionID)
	if err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

func respondJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func respondErr(w http.ResponseWriter, status int, msg string) {
	respondJSON(w, status, map[string]string{"error": msg})
}
