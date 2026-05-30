package portforward

import "github.com/xhd2015/agent-pro/agent/exec/tool_resolve"

// IsCommandAvailable checks if a command is available on PATH
func IsCommandAvailable(name string) bool {
	return tool_resolve.IsAvailable(name)
}
