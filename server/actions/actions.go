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

// ActionRunRequest represents a request to run an action
type ActionRunRequest struct {
	ProjectDir string `json:"project_dir"`
	Script     string `json:"script"`
}

var (
	baseDir = config.ProjectsDir
	mu      sync.RWMutex
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
	sw := sse.NewWriter(w)
	if sw == nil {
		return fmt.Errorf("streaming not supported")
	}

	sw.SendLog(fmt.Sprintf("Running action in %s...", projectDir))

	// Create command using shell
	var cmd *exec.Cmd
	if strings.Contains(script, "\n") || strings.Contains(script, ";") || strings.Contains(script, "&&") {
		// Multi-line or complex command - use shell
		cmd = exec.Command("bash", "-c", script)
	} else {
		// Simple command - parse and execute directly
		parts := strings.Fields(script)
		if len(parts) == 0 {
			sw.SendError("Empty script")
			sw.SendDone(map[string]string{"success": "false", "message": "Empty script"})
			return nil
		}
		cmd = exec.Command(parts[0], parts[1:]...)
	}

	cmd.Dir = projectDir

	// Set up environment with PATH additions
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	cmd.Env = tool_resolve.AppendExtraPaths(cmd.Env)

	// Stream the command output
	err := sw.StreamCmd(cmd)
	if err != nil {
		sw.SendError(fmt.Sprintf("Action failed: %v", err))
		sw.SendDone(map[string]string{"success": "false", "message": err.Error()})
		return nil
	}

	sw.SendLog("Action completed successfully")
	sw.SendDone(map[string]string{"success": "true", "message": "Action completed successfully"})
	return nil
}

// RegisterAPI registers the actions API endpoints
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/actions", handleActions)
	mux.HandleFunc("/api/actions/", handleActionByID)
	mux.HandleFunc("/api/actions/run", handleRunAction)
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

func respondJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func respondErr(w http.ResponseWriter, status int, msg string) {
	respondJSON(w, status, map[string]string{"error": msg})
}
