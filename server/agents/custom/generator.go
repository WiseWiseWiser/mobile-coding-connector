package custom

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func GenerateOpencodeConfig(agentID string) error {
	agent, err := LoadAgent(agentID)
	if err != nil {
		return err
	}
	if agent == nil {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	cfg := AgentConfig{}
	if agent.Name != "" {
		cfg.Name = agent.Name
	}
	if agent.Description != "" {
		cfg.Description = agent.Description
	}
	cfg.Mode = agent.Mode
	cfg.Tools = agent.Tools
	cfg.Permissions = agent.Permissions
	if agent.Model != "" {
		cfg.Model = agent.Model
	}

	genDir := OpencodeGenDir(agentID)
	if err := os.MkdirAll(genDir, 0755); err != nil {
		return err
	}

	systemPromptPath := SystemPromptPath(agentID)
	hasSystemPrompt := false
	if _, err := os.Stat(systemPromptPath); err == nil {
		hasSystemPrompt = true
	}

	opencodeCfg := make(map[string]interface{})
	agentCfg := make(map[string]interface{})

	if cfg.Description != "" {
		agentCfg["description"] = cfg.Description
	}

	if cfg.Mode != "" {
		agentCfg["mode"] = cfg.Mode
	}

	if cfg.Model != "" {
		agentCfg["model"] = cfg.Model
	}

	if hasSystemPrompt {
		relativePath, err := filepath.Rel(genDir, systemPromptPath)
		if err != nil {
			relativePath = systemPromptPath
		}
		agentCfg["prompt"] = "{file:" + relativePath + "}"
	}

	if len(cfg.Tools) > 0 {
		agentCfg["tools"] = cfg.Tools
	}

	if len(cfg.Permissions) > 0 {
		agentCfg["permissions"] = cfg.Permissions
	}

	opencodeCfg["agent"] = map[string]interface{}{
		agentID: agentCfg,
	}
	opencodeCfg["default_agent"] = agentID

	data, err := json.MarshalIndent(opencodeCfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(OpencodeConfigPath(agentID), data, 0644)
}

func GetOpencodeConfigDir(agentID string) string {
	return OpencodeGenDir(agentID)
}
