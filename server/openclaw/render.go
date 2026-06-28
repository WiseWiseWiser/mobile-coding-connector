package openclaw

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type secretRef struct {
	Source   string `json:"source"`
	Provider string `json:"provider"`
	ID       string `json:"id"`
}

type renderedSlackChannel struct {
	Enabled        bool       `json:"enabled"`
	Mode           string     `json:"mode"`
	AppToken       secretRef  `json:"appToken"`
	BotToken       secretRef  `json:"botToken"`
	DMPolicy       string     `json:"dmPolicy,omitempty"`
	AllowFrom      []string   `json:"allowFrom,omitempty"`
	Groups         any        `json:"groups,omitempty"`
}

type renderedAgentsDefaults struct {
	Workspace string            `json:"workspace,omitempty"`
	Model     *renderedModel    `json:"model,omitempty"`
}

type renderedModel struct {
	Primary string `json:"primary"`
}

type renderedGateway struct {
	Port int `json:"port"`
}

type renderedConfig struct {
	Gateway renderedGateway       `json:"gateway"`
	Agents  renderedAgentsBlock   `json:"agents"`
	Channels renderedChannelsBlock `json:"channels,omitempty"`
}

type renderedAgentsBlock struct {
	Defaults renderedAgentsDefaults `json:"defaults"`
}

type renderedChannelsBlock struct {
	Slack renderedSlackChannel `json:"slack"`
}

func RenderGatewayConfig(cfg *Config) ([]byte, error) {
	if cfg == nil {
		cfg = defaultConfig()
	}
	normalizeConfig(cfg)

	out := renderedConfig{
		Gateway: renderedGateway{Port: cfg.GatewayPort},
		Agents: renderedAgentsBlock{
			Defaults: renderedAgentsDefaults{},
		},
	}

	if workspace := cfg.Workspace; workspace != "" {
		out.Agents.Defaults.Workspace = workspace
	}
	if model := cfg.Model; model != "" {
		out.Agents.Defaults.Model = &renderedModel{Primary: model}
	}

	if cfg.Slack != nil && cfg.Slack.Enabled {
		slack := renderedSlackChannel{
			Enabled: true,
			Mode:    slackModeSocket,
			AppToken: secretRef{
				Source:   "env",
				Provider: "default",
				ID:       "SLACK_APP_TOKEN",
			},
			BotToken: secretRef{
				Source:   "env",
				Provider: "default",
				ID:       "SLACK_BOT_TOKEN",
			},
			DMPolicy:  cfg.Slack.DMPolicy,
			AllowFrom: cfg.Slack.AllowFrom,
		}
		if cfg.Slack.RequireMention != nil && *cfg.Slack.RequireMention {
			slack.Groups = map[string]map[string]bool{
				"*": {"requireMention": true},
			}
		}
		out.Channels = renderedChannelsBlock{Slack: slack}
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal rendered config: %w", err)
	}
	return append(data, '\n'), nil
}

func WriteGeneratedConfig(cfg *Config) (string, error) {
	data, err := RenderGatewayConfig(cfg)
	if err != nil {
		return "", err
	}
	path := generatedConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", fmt.Errorf("failed to create openclaw dir: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write generated config: %w", err)
	}
	return path, nil
}