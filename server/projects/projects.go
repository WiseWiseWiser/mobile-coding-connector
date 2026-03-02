package projects

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
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
	"github.com/xhd2015/lifelog-private/ai-critic/server/gitrunner"
)

var projectsFile = config.ProjectsFile

type Todo struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Done      bool   `json:"done"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type Project struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	RepoURL   string `json:"repo_url"`
	Dir       string `json:"dir"`
	SSHKeyID  string `json:"ssh_key_id,omitempty"`
	UseSSH    bool   `json:"use_ssh"`
	CreatedAt string `json:"created_at"`
	ParentID  string `json:"parent_id,omitempty"`
	Todos     []Todo `json:"todos,omitempty"`
	Readme    string `json:"readme,omitempty"`
}

// GitStatusInfo holds git status information for a project
type GitStatusInfo struct {
	IsClean     bool `json:"is_clean"`
	Uncommitted int  `json:"uncommitted"`
}

func getGitStatus(projectDir string) GitStatusInfo {
	// Check if it's a git repository
	if err := gitrunner.RevParse("--git-dir").Dir(projectDir).RunSilent(); err != nil {
		return GitStatusInfo{IsClean: true, Uncommitted: 0}
	}

	// Get status with porcelain format
	output, err := gitrunner.Status("--porcelain=v1").Dir(projectDir).Output()
	if err != nil {
		return GitStatusInfo{IsClean: true, Uncommitted: 0}
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	count := 0
	for _, line := range lines {
		if len(line) >= 2 {
			count++
		}
	}

	return GitStatusInfo{
		IsClean:     count == 0,
		Uncommitted: count,
	}
}

var mu sync.RWMutex

func ensureDir() error {
	return os.MkdirAll(filepath.Dir(projectsFile), 0755)
}

func loadAll() ([]Project, error) {
	data, err := os.ReadFile(projectsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []Project{}, nil
		}
		return nil, err
	}
	var list []Project
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func saveAll(list []Project) error {
	if err := ensureDir(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(projectsFile, data, 0644)
}

func Add(p Project) (string, error) {
	mu.Lock()
	defer mu.Unlock()
	list, err := loadAll()
	if err != nil {
		return "", err
	}
	for _, existing := range list {
		if existing.Dir == p.Dir {
			return existing.ID, nil
		}
	}
	if p.ID == "" {
		p.ID = fmt.Sprintf("%d", time.Now().UnixMilli())
	}
	if p.CreatedAt == "" {
		p.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	list = append(list, p)
	if err := saveAll(list); err != nil {
		return "", err
	}
	return p.ID, nil
}

func List() ([]Project, error) {
	mu.RLock()
	defer mu.RUnlock()
	return loadAll()
}

func Remove(id string) error {
	mu.Lock()
	defer mu.Unlock()
	list, err := loadAll()
	if err != nil {
		return err
	}
	filtered := make([]Project, 0, len(list))
	found := false
	for _, p := range list {
		if p.ID == id {
			found = true
			continue
		}
		if p.ParentID == id {
			p.ParentID = ""
		}
		filtered = append(filtered, p)
	}
	if !found {
		return fmt.Errorf("project not found: %s", id)
	}
	return saveAll(filtered)
}

// ProjectUpdate contains the fields that can be updated.
// Pointer fields: nil means "no change", non-nil means "set to this value" (empty string means "unset").
type ProjectUpdate struct {
	SSHKeyID *string `json:"ssh_key_id"`
	UseSSH   *bool   `json:"use_ssh"`
	ParentID *string `json:"parent_id"`
	Readme   *string `json:"readme"`
}

func Update(id string, updates ProjectUpdate) (*Project, error) {
	mu.Lock()
	defer mu.Unlock()
	list, err := loadAll()
	if err != nil {
		return nil, err
	}
	for i, p := range list {
		if p.ID != id {
			continue
		}
		if updates.SSHKeyID != nil {
			list[i].SSHKeyID = *updates.SSHKeyID
		}
		if updates.UseSSH != nil {
			list[i].UseSSH = *updates.UseSSH
		}
		if updates.ParentID != nil {
			list[i].ParentID = *updates.ParentID
		}
		if updates.Readme != nil {
			list[i].Readme = *updates.Readme
		}
		if err := saveAll(list); err != nil {
			return nil, err
		}
		return &list[i], nil
	}
	return nil, fmt.Errorf("project not found: %s", id)
}

func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/projects", handleProjects)
	mux.HandleFunc("/api/projects/todos", handleTodos)
	mux.HandleFunc("/api/projects/readme", handleReadme)
}

func handleReadme(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		respondErr(w, http.StatusBadRequest, "project_id is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		mu.RLock()
		defer mu.RUnlock()
		list, err := loadAll()
		if err != nil {
			respondErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		for _, p := range list {
			if p.ID == projectID {
				respondJSON(w, http.StatusOK, map[string]string{"readme": p.Readme})
				return
			}
		}
		respondErr(w, http.StatusNotFound, "project not found")

	case http.MethodPut:
		var req struct {
			Readme string `json:"readme"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondErr(w, http.StatusBadRequest, "invalid request body")
			return
		}

		mu.Lock()
		defer mu.Unlock()
		list, err := loadAll()
		if err != nil {
			respondErr(w, http.StatusInternalServerError, err.Error())
			return
		}

		var projectIndex int = -1
		for i, p := range list {
			if p.ID == projectID {
				projectIndex = i
				break
			}
		}

		if projectIndex == -1 {
			respondErr(w, http.StatusNotFound, "project not found")
			return
		}

		list[projectIndex].Readme = req.Readme
		if err := saveAll(list); err != nil {
			respondErr(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleProjects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := List()
		if err != nil {
			respondErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		parentID := r.URL.Query().Get("parent_id")
		includeAll := r.URL.Query().Get("all") == "true"
		type ProjectWithStatus struct {
			Project
			DirExists bool          `json:"dir_exists"`
			GitStatus GitStatusInfo `json:"git_status"`
		}
		var filtered []Project
		if includeAll {
			filtered = list
		} else {
			filtered = make([]Project, 0, len(list))
			for _, p := range list {
				if parentID == "" {
					if p.ParentID == "" {
						filtered = append(filtered, p)
					}
				} else if p.ParentID == parentID {
					filtered = append(filtered, p)
				}
			}
		}
		result := make([]ProjectWithStatus, len(filtered))
		for i, p := range filtered {
			_, statErr := os.Stat(p.Dir)
			gitStatus := getGitStatus(p.Dir)
			result[i] = ProjectWithStatus{
				Project:   p,
				DirExists: statErr == nil,
				GitStatus: gitStatus,
			}
		}
		respondJSON(w, http.StatusOK, result)
	case http.MethodPost:
		var req struct {
			Name     string `json:"name"`
			Dir      string `json:"dir"`
			ParentID string `json:"parent_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondErr(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Dir == "" {
			respondErr(w, http.StatusBadRequest, "dir is required")
			return
		}
		// Resolve to absolute path
		absDir, err := filepath.Abs(req.Dir)
		if err != nil {
			respondErr(w, http.StatusBadRequest, fmt.Sprintf("invalid dir: %v", err))
			return
		}
		// Verify directory exists
		info, err := os.Stat(absDir)
		if err != nil {
			respondErr(w, http.StatusBadRequest, fmt.Sprintf("cannot access dir: %v", err))
			return
		}
		if !info.IsDir() {
			respondErr(w, http.StatusBadRequest, "path is not a directory")
			return
		}
		// Use directory basename as name if not provided
		name := req.Name
		if name == "" {
			name = filepath.Base(absDir)
		}
		p := Project{
			Name:     name,
			Dir:      absDir,
			ParentID: req.ParentID,
		}
		projectID, err := Add(p)
		if err != nil {
			respondErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok", "id": projectID, "dir": absDir, "name": name})
	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			respondErr(w, http.StatusBadRequest, "id is required")
			return
		}
		if err := Remove(id); err != nil {
			respondErr(w, http.StatusNotFound, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	case http.MethodPatch:
		id := r.URL.Query().Get("id")
		if id == "" {
			respondErr(w, http.StatusBadRequest, "id is required")
			return
		}
		var updates ProjectUpdate
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			respondErr(w, http.StatusBadRequest, "invalid request body")
			return
		}
		project, err := Update(id, updates)
		if err != nil {
			respondErr(w, http.StatusNotFound, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, project)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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

func handleTodos(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		respondErr(w, http.StatusBadRequest, "project_id is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		list, err := List()
		if err != nil {
			respondErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		for _, p := range list {
			if p.ID == projectID {
				respondJSON(w, http.StatusOK, p.Todos)
				return
			}
		}
		respondErr(w, http.StatusNotFound, "project not found")

	case http.MethodPost:
		var req struct {
			Text string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondErr(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Text == "" {
			respondErr(w, http.StatusBadRequest, "text is required")
			return
		}

		mu.Lock()
		defer mu.Unlock()
		list, err := loadAll()
		if err != nil {
			respondErr(w, http.StatusInternalServerError, err.Error())
			return
		}

		var projectIndex int = -1
		for i, p := range list {
			if p.ID == projectID {
				projectIndex = i
				break
			}
		}

		if projectIndex == -1 {
			respondErr(w, http.StatusNotFound, "project not found")
			return
		}

		todo := Todo{
			ID:        fmt.Sprintf("%d", time.Now().UnixMilli()),
			Text:      req.Text,
			Done:      false,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}

		list[projectIndex].Todos = append(list[projectIndex].Todos, todo)
		if err := saveAll(list); err != nil {
			respondErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, todo)

	case http.MethodPut:
		var req struct {
			ID   string  `json:"id"`
			Text *string `json:"text,omitempty"`
			Done *bool   `json:"done,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondErr(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.ID == "" {
			respondErr(w, http.StatusBadRequest, "id is required")
			return
		}

		mu.Lock()
		defer mu.Unlock()
		list, err := loadAll()
		if err != nil {
			respondErr(w, http.StatusInternalServerError, err.Error())
			return
		}

		var projectIndex int = -1
		var todoIndex int = -1
		for i, p := range list {
			if p.ID == projectID {
				projectIndex = i
				for j, t := range p.Todos {
					if t.ID == req.ID {
						todoIndex = j
						break
					}
				}
				break
			}
		}

		if projectIndex == -1 {
			respondErr(w, http.StatusNotFound, "project not found")
			return
		}
		if todoIndex == -1 {
			respondErr(w, http.StatusNotFound, "todo not found")
			return
		}

		if req.Text != nil {
			list[projectIndex].Todos[todoIndex].Text = *req.Text
			list[projectIndex].Todos[todoIndex].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		}
		if req.Done != nil {
			list[projectIndex].Todos[todoIndex].Done = *req.Done
		}

		if err := saveAll(list); err != nil {
			respondErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, list[projectIndex].Todos[todoIndex])

	case http.MethodDelete:
		todoID := r.URL.Query().Get("todo_id")
		if todoID == "" {
			respondErr(w, http.StatusBadRequest, "todo_id is required")
			return
		}

		mu.Lock()
		defer mu.Unlock()
		list, err := loadAll()
		if err != nil {
			respondErr(w, http.StatusInternalServerError, err.Error())
			return
		}

		var projectIndex int = -1
		for i, p := range list {
			if p.ID == projectID {
				projectIndex = i
				break
			}
		}

		if projectIndex == -1 {
			respondErr(w, http.StatusNotFound, "project not found")
			return
		}

		filteredTodos := make([]Todo, 0, len(list[projectIndex].Todos))
		for _, t := range list[projectIndex].Todos {
			if t.ID != todoID {
				filteredTodos = append(filteredTodos, t)
			}
		}

		if len(filteredTodos) == len(list[projectIndex].Todos) {
			respondErr(w, http.StatusNotFound, "todo not found")
			return
		}

		list[projectIndex].Todos = filteredTodos
		if err := saveAll(list); err != nil {
			respondErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// WorktreeInfo holds information about a git worktree
type WorktreeInfo struct {
	Path      string `json:"path"`
	Branch    string `json:"branch"`
	WorktreeID int    `json:"worktreeId"`
}

// ResolveProjectDir resolves the actual project directory based on project name and optional worktree ID.
// It looks up the project in the project list and returns its directory.
// If worktreeID is provided, it resolves the correct worktree directory.
func ResolveProjectDir(projectName, worktreeID string) (string, error) {
	projects, err := List()
	if err != nil {
		return "", fmt.Errorf("failed to list projects: %w", err)
	}

	// Find the project by name
	var projectDir string
	for _, p := range projects {
		if p.Name == projectName {
			projectDir = p.Dir
			break
		}
	}

	if projectDir == "" {
		return "", fmt.Errorf("project not found: %s", projectName)
	}

	// If no worktree ID specified, return the project directory directly
	if worktreeID == "" {
		return projectDir, nil
	}

	// Parse worktree ID
	worktreeIDInt, err := strconv.Atoi(worktreeID)
	if err != nil {
		return "", fmt.Errorf("invalid worktree ID: %s", worktreeID)
	}

	// Get worktrees for the project
	worktrees, err := GetWorktreesForProject(projectDir)
	if err != nil {
		return "", fmt.Errorf("failed to get worktrees: %w", err)
	}

	// Find the worktree with matching ID
	for _, wt := range worktrees {
		if wt.WorktreeID == worktreeIDInt {
			return wt.Path, nil
		}
	}

	return "", fmt.Errorf("worktree not found: %d", worktreeIDInt)
}

// GetWorktreesForProject returns all worktrees for a given git repository
func GetWorktreesForProject(repoDir string) ([]WorktreeInfo, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	return ParseWorktreesOutput(string(output)), nil
}

// ParseWorktreesOutput parses the output of `git worktree list --porcelain`
func ParseWorktreesOutput(output string) []WorktreeInfo {
	var worktrees []WorktreeInfo
	var current *WorktreeInfo

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			if current != nil {
				worktrees = append(worktrees, *current)
			}
			path := strings.TrimPrefix(line, "worktree ")
			current = &WorktreeInfo{
				Path: path,
			}
		} else if strings.HasPrefix(line, "branch ") && current != nil {
			branch := strings.TrimPrefix(line, "branch ")
			// Extract just the branch name (remove refs/heads/ prefix)
			branch = strings.TrimPrefix(branch, "refs/heads/")
			current.Branch = branch
		}
	}

	// Don't forget the last one
	if current != nil {
		worktrees = append(worktrees, *current)
	}

	// Assign worktree IDs (main is always 0, others start from 1)
	for i := range worktrees {
		worktrees[i].WorktreeID = i
	}

	return worktrees
}
