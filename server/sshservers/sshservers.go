package sshservers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
)

// SSHServer represents a user-configured SSH server connection
type SSHServer struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Username  string `json:"username"`
	SSHKeyID  string `json:"ssh_key_id"`
	CreatedAt string `json:"created_at"`
}

var (
	serversFile = config.SSHServerFile
	mu          sync.RWMutex
)

// loadServers reads the SSH servers from disk
func loadServers() ([]SSHServer, error) {
	mu.RLock()
	defer mu.RUnlock()

	data, err := os.ReadFile(serversFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []SSHServer{}, nil
		}
		return nil, err
	}

	var servers []SSHServer
	if err := json.Unmarshal(data, &servers); err != nil {
		return nil, err
	}

	return servers, nil
}

// saveServers writes the SSH servers to disk
func saveServers(servers []SSHServer) error {
	mu.Lock()
	defer mu.Unlock()

	// Ensure directory exists
	dir := config.DataDir
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(servers, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(serversFile, data, 0644)
}

// ListServers returns all configured SSH servers
func ListServers() ([]SSHServer, error) {
	return loadServers()
}

// AddServer adds a new SSH server
func AddServer(server SSHServer) (SSHServer, error) {
	servers, err := loadServers()
	if err != nil {
		return SSHServer{}, err
	}

	// Generate ID if not provided
	if server.ID == "" {
		server.ID = fmt.Sprintf("ssh-server-%d", time.Now().UnixMilli())
	}
	server.CreatedAt = time.Now().UTC().Format(time.RFC3339)

	servers = append(servers, server)
	if err := saveServers(servers); err != nil {
		return SSHServer{}, err
	}

	return server, nil
}

// UpdateServer updates an existing SSH server
func UpdateServer(id string, updates SSHServer) (SSHServer, error) {
	servers, err := loadServers()
	if err != nil {
		return SSHServer{}, err
	}

	for i, s := range servers {
		if s.ID == id {
			servers[i] = SSHServer{
				ID:        s.ID,
				Name:      updates.Name,
				Host:      updates.Host,
				Port:      updates.Port,
				Username:  updates.Username,
				SSHKeyID:  updates.SSHKeyID,
				CreatedAt: s.CreatedAt,
			}
			if err := saveServers(servers); err != nil {
				return SSHServer{}, err
			}
			return servers[i], nil
		}
	}

	return SSHServer{}, fmt.Errorf("server not found")
}

// DeleteServer removes an SSH server by ID
func DeleteServer(id string) error {
	servers, err := loadServers()
	if err != nil {
		return err
	}

	for i, s := range servers {
		if s.ID == id {
			servers = append(servers[:i], servers[i+1:]...)
			return saveServers(servers)
		}
	}

	return fmt.Errorf("server not found")
}

// GetServer returns a single server by ID
func GetServer(id string) (SSHServer, error) {
	servers, err := loadServers()
	if err != nil {
		return SSHServer{}, err
	}

	for _, s := range servers {
		if s.ID == id {
			return s, nil
		}
	}

	return SSHServer{}, fmt.Errorf("server not found")
}

// HTTP Handlers

// RegisterAPI registers the SSH server API endpoints
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/ssh-servers", handleServers)
	mux.HandleFunc("/api/ssh-servers/", handleServerByID)
}

func handleServers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		servers, err := ListServers()
		if err != nil {
			respondErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, servers)

	case http.MethodPost:
		var server SSHServer
		if err := json.NewDecoder(r.Body).Decode(&server); err != nil {
			respondErr(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if server.Name == "" || server.Host == "" || server.Username == "" {
			respondErr(w, http.StatusBadRequest, "name, host, and username are required")
			return
		}

		result, err := AddServer(server)
		if err != nil {
			respondErr(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusCreated, result)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleServerByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/ssh-servers/")
	parts := strings.Split(path, "/")
	if len(parts) < 1 || parts[0] == "" {
		respondErr(w, http.StatusBadRequest, "server ID is required")
		return
	}
	id := parts[0]

	switch r.Method {
	case http.MethodGet:
		server, err := GetServer(id)
		if err != nil {
			respondErr(w, http.StatusNotFound, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, server)

	case http.MethodPut:
		var server SSHServer
		if err := json.NewDecoder(r.Body).Decode(&server); err != nil {
			respondErr(w, http.StatusBadRequest, "invalid request body")
			return
		}

		result, err := UpdateServer(id, server)
		if err != nil {
			if err.Error() == "server not found" {
				respondErr(w, http.StatusNotFound, err.Error())
			} else {
				respondErr(w, http.StatusInternalServerError, err.Error())
			}
			return
		}

		respondJSON(w, http.StatusOK, result)

	case http.MethodDelete:
		if err := DeleteServer(id); err != nil {
			if err.Error() == "server not found" {
				respondErr(w, http.StatusNotFound, err.Error())
			} else {
				respondErr(w, http.StatusInternalServerError, err.Error())
			}
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})

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
