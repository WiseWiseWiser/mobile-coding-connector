package cursor_acp

import (
	"encoding/json"
	"os"
)

const settingsFile = ".ai-critic/cursor-agent.json"

type CursorAgentSettings struct {
	APIKey     string `json:"api_key,omitempty"`
	BinaryPath string `json:"binary_path,omitempty"`
}

type EffectivePathInfo struct {
	Found         bool   `json:"found"`
	EffectivePath string `json:"effective_path,omitempty"`
	Error         string `json:"error,omitempty"`
}

func ResolveEffectivePath() EffectivePathInfo {
	path, err := resolveAgentPath()
	if err != nil {
		return EffectivePathInfo{Found: false, Error: err.Error()}
	}
	return EffectivePathInfo{Found: true, EffectivePath: path}
}

func LoadSettings() (*CursorAgentSettings, error) {
	data, err := os.ReadFile(settingsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &CursorAgentSettings{}, nil
		}
		return nil, err
	}
	var s CursorAgentSettings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func SaveSettings(s *CursorAgentSettings) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	os.MkdirAll(".ai-critic", 0755)
	return os.WriteFile(settingsFile, data, 0644)
}
