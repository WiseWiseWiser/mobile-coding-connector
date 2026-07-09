package menubar

import (
	"strings"
	"time"
)

// PeriodicRefreshInterval is the app-side poll period for services + terminals.
const PeriodicRefreshInterval = 30 * time.Second

// FormatTerminalTitle returns the Terminals submenu title for a session.
// Non-empty trimmed name wins; empty/whitespace name falls back to id.
// When status is "exited" (case-insensitive, trimmed), appends " [EXITED]".
func FormatTerminalTitle(name, id, status string) string {
	base := id
	if strings.TrimSpace(name) != "" {
		base = name
	}
	if strings.EqualFold(strings.TrimSpace(status), "exited") {
		return base + " [EXITED]"
	}
	return base
}

// FormatTerminalsEmptyLabel is shown when the sessions list is empty.
func FormatTerminalsEmptyLabel() string {
	return "No terminal sessions"
}

// BuildTerminalAttachCommand builds the CLI line to attach to a session by id.
func BuildTerminalAttachCommand(agentBinary, sessionID string) string {
	return agentBinary + " terminal attach " + sessionID
}

// BuildTerminalNewCommand builds the CLI line to open a new terminal session.
func BuildTerminalNewCommand(agentBinary string) string {
	return agentBinary + " terminal new"
}

// AgentBinaryForApp returns the agent CLI name for local vs remote menu-bar apps.
func AgentBinaryForApp(isRemote bool) string {
	if isRemote {
		return "remote-agent"
	}
	return "local-agent"
}
