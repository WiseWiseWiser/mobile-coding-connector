package projects

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const projectsFile = ".ai-critic/projects.json"

type Project struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	RepoURL   string `json:"repo_url"`
	Dir       string `json:"dir"`
	SSHKeyID  string `json:"ssh_key_id,omitempty"`
	UseSSH    bool   `json:"use_ssh"`
	CreatedAt string `json:"created_at"`
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

func Add(p Project) error {
	mu.Lock()
	defer mu.Unlock()
	list, err := loadAll()
	if err != nil {
		return err
	}
	if p.ID == "" {
		p.ID = fmt.Sprintf("%d", time.Now().UnixMilli())
	}
	if p.CreatedAt == "" {
		p.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	list = append(list, p)
	return saveAll(list)
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
		if err := saveAll(list); err != nil {
			return nil, err
		}
		return &list[i], nil
	}
	return nil, fmt.Errorf("project not found: %s", id)
}

func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/projects", handleProjects)
}

func handleProjects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := List()
		if err != nil {
			respondErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, list)
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
