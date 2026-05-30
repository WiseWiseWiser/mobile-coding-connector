package settings

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xhd2015/ai-critic/server/config"
)

// GitUserConfig is one Git author/committer identity exposed by the Git
// Config tab and the remote-agent settings command.
type GitUserConfig struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	CreatedAt string `json:"createdAt"`
}

type gitUserConfigRequest struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Email     string          `json:"email"`
	CreatedAt string          `json:"createdAt"`
	Configs   []GitUserConfig `json:"configs"`
}

func handleGitUserConfigs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		configs, err := LoadGitUserConfigs()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, configs)
	case http.MethodPost:
		var req gitUserConfigRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		config, err := AddGitUserConfig(req.ID, req.Name, req.Email, req.CreatedAt)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, config)
	case http.MethodPatch:
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			writeJSONError(w, http.StatusBadRequest, "id is required")
			return
		}
		var req gitUserConfigRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		config, err := UpdateGitUserConfig(id, req.Name, req.Email)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, config)
	case http.MethodPut:
		var req gitUserConfigRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		configs, err := SaveGitUserConfigs(req.Configs)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, configs)
	case http.MethodDelete:
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			writeJSONError(w, http.StatusBadRequest, "id is required")
			return
		}
		if err := DeleteGitUserConfig(id); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, map[string]string{"status": "ok"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func LoadGitUserConfigs() ([]GitUserConfig, error) {
	data, err := os.ReadFile(config.GitUserConfigsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []GitUserConfig{}, nil
		}
		return nil, fmt.Errorf("read git user configs: %w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return []GitUserConfig{}, nil
	}

	var configs []GitUserConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, fmt.Errorf("parse git user configs: %w", err)
	}
	return normalizeGitUserConfigs(configs)
}

func SaveGitUserConfigs(configs []GitUserConfig) ([]GitUserConfig, error) {
	normalized, err := normalizeGitUserConfigs(configs)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(config.GitUserConfigsFile), 0755); err != nil {
		return nil, fmt.Errorf("create settings dir: %w", err)
	}
	data, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal git user configs: %w", err)
	}
	if err := os.WriteFile(config.GitUserConfigsFile, data, 0644); err != nil {
		return nil, fmt.Errorf("write git user configs: %w", err)
	}
	return normalized, nil
}

func AddGitUserConfig(id string, name string, email string, createdAt string) (*GitUserConfig, error) {
	configs, err := LoadGitUserConfigs()
	if err != nil {
		return nil, err
	}
	config, err := normalizeGitUserConfig(GitUserConfig{
		ID:        id,
		Name:      name,
		Email:     email,
		CreatedAt: createdAt,
	})
	if err != nil {
		return nil, err
	}
	if config.ID == "" {
		config.ID = newGitUserConfigID()
	}
	for _, existing := range configs {
		if existing.ID == config.ID {
			return nil, fmt.Errorf("git user config id already exists: %s", config.ID)
		}
	}
	configs = append(configs, *config)
	if _, err := SaveGitUserConfigs(configs); err != nil {
		return nil, err
	}
	return config, nil
}

func UpdateGitUserConfig(id string, name string, email string) (*GitUserConfig, error) {
	configs, err := LoadGitUserConfigs()
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	for i := range configs {
		if configs[i].ID != id {
			continue
		}
		updated, err := normalizeGitUserConfig(GitUserConfig{
			ID:        configs[i].ID,
			Name:      name,
			Email:     email,
			CreatedAt: configs[i].CreatedAt,
		})
		if err != nil {
			return nil, err
		}
		configs[i] = *updated
		if _, err := SaveGitUserConfigs(configs); err != nil {
			return nil, err
		}
		return updated, nil
	}
	return nil, fmt.Errorf("git user config not found: %s", id)
}

func DeleteGitUserConfig(id string) error {
	configs, err := LoadGitUserConfigs()
	if err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	next := configs[:0]
	found := false
	for _, config := range configs {
		if config.ID == id {
			found = true
			continue
		}
		next = append(next, config)
	}
	if !found {
		return fmt.Errorf("git user config not found: %s", id)
	}
	_, err = SaveGitUserConfigs(next)
	return err
}

func normalizeGitUserConfigs(configs []GitUserConfig) ([]GitUserConfig, error) {
	seen := make(map[string]bool, len(configs))
	normalized := make([]GitUserConfig, 0, len(configs))
	for _, config := range configs {
		item, err := normalizeGitUserConfig(config)
		if err != nil {
			return nil, err
		}
		if item.ID == "" {
			item.ID = stableGitUserConfigID(item.Name, item.Email)
		}
		if seen[item.ID] {
			continue
		}
		seen[item.ID] = true
		normalized = append(normalized, *item)
	}
	return normalized, nil
}

func normalizeGitUserConfig(config GitUserConfig) (*GitUserConfig, error) {
	name := strings.TrimSpace(config.Name)
	email := strings.TrimSpace(config.Email)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}
	createdAt := strings.TrimSpace(config.CreatedAt)
	if createdAt == "" {
		createdAt = time.Now().UTC().Format(time.RFC3339)
	}
	return &GitUserConfig{
		ID:        strings.TrimSpace(config.ID),
		Name:      name,
		Email:     email,
		CreatedAt: createdAt,
	}, nil
}

func stableGitUserConfigID(name string, email string) string {
	hash := uint32(5381)
	for _, r := range name + "\n" + email {
		hash = ((hash << 5) + hash) ^ uint32(r)
	}
	return "git-user-" + strconv.FormatUint(uint64(hash), 36)
}

func newGitUserConfigID() string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err == nil {
		return fmt.Sprintf("%s-%s", time.Now().UTC().Format("20060102150405"), hex.EncodeToString(b[:]))
	}
	return fmt.Sprintf("%d", time.Now().UTC().UnixNano())
}
