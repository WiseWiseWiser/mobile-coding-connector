package custom

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
)

const AgentsDirName = "agents"

func GetCustomAgentsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".ai-critic", AgentsDirName)
}

var (
	CustomAgentsDir = GetCustomAgentsDir()
)

type AgentConfig struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Mode        string            `json:"mode"` // "primary" or "subagent"
	Model       string            `json:"model,omitempty"`
	Tools       map[string]bool   `json:"tools"`
	Permissions map[string]string `json:"permissions,omitempty"`
}

type CustomAgent struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Mode            string            `json:"mode"`
	Model           string            `json:"model,omitempty"`
	Tools           map[string]bool   `json:"tools"`
	Permissions     map[string]string `json:"permissions,omitempty"`
	HasSystemPrompt bool              `json:"hasSystemPrompt"`
}

func (a *AgentConfig) Validate() error {
	if a.Name == "" {
		return nil // will use ID as name
	}
	if a.Mode != "primary" && a.Mode != "subagent" {
		a.Mode = "primary"
	}
	return nil
}

func AgentDir(agentID string) string {
	dir := GetCustomAgentsDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, agentID)
}

func AgentConfigPath(agentID string) string {
	return filepath.Join(AgentDir(agentID), "agent.json")
}

func SystemPromptPath(agentID string) string {
	return filepath.Join(AgentDir(agentID), "SYSTEM_PROMPT.md")
}

func OpencodeGenDir(agentID string) string {
	return filepath.Join(AgentDir(agentID), "opencode-gen")
}

func OpencodeConfigPath(agentID string) string {
	return filepath.Join(OpencodeGenDir(agentID), "opencode.json")
}

func GetAgentsDir() string {
	return CustomAgentsDir
}

func EnsureAgentsDir() error {
	dir := GetAgentsDir()
	if dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0755)
}

func ListAgents() ([]CustomAgent, error) {
	dir := GetAgentsDir()
	if dir == "" {
		return []CustomAgent{}, nil
	}

	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []CustomAgent{}, nil
	}
	if err != nil {
		return nil, err
	}

	var agents []CustomAgent
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		agentID := entry.Name()
		agent, err := LoadAgent(agentID)
		if err != nil {
			continue
		}
		if agent != nil {
			agents = append(agents, *agent)
		}
	}

	slices.SortFunc(agents, func(a, b CustomAgent) int {
		return compareAgentID(a.ID, b.ID)
	})
	return agents, nil
}

func LoadAgent(agentID string) (*CustomAgent, error) {
	configPath := AgentConfigPath(agentID)
	data, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var cfg AgentConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	hasSystemPrompt := false
	if _, err := os.Stat(SystemPromptPath(agentID)); err == nil {
		hasSystemPrompt = true
	}

	name := cfg.Name
	if name == "" {
		name = agentID
	}

	return &CustomAgent{
		ID:              agentID,
		Name:            name,
		Description:     cfg.Description,
		Mode:            cfg.Mode,
		Model:           cfg.Model,
		Tools:           cfg.Tools,
		Permissions:     cfg.Permissions,
		HasSystemPrompt: hasSystemPrompt,
	}, nil
}

func SaveAgent(agentID string, cfg *AgentConfig) error {
	agentDir := AgentDir(agentID)
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		return err
	}

	cfg.Name = toTitleCase(cfg.Name)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(AgentConfigPath(agentID), data, 0644)
}

func SaveSystemPrompt(agentID string, content string) error {
	agentDir := AgentDir(agentID)
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(SystemPromptPath(agentID), []byte(content), 0644)
}

func DeleteAgent(agentID string) error {
	agentDir := AgentDir(agentID)
	return os.RemoveAll(agentDir)
}

func GetSystemPrompt(agentID string) (string, error) {
	data, err := os.ReadFile(SystemPromptPath(agentID))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func compareAgentID(a, b string) int {
	order := []string{"build", "plan", "refactor", "debug"}
	ai := len(order)
	bi := len(order)
	for i, o := range order {
		if a == o {
			ai = i
		}
		if b == o {
			bi = i
		}
	}
	if ai != bi {
		return ai - bi
	}
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func toTitleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	runes := []rune(s)
	runes[0] = toUpper(runes[0])
	return string(runes)
}

func toUpper(r rune) rune {
	if r >= 'a' && r <= 'z' {
		return r - 32
	}
	return r
}
