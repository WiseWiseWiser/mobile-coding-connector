package portforward

import "github.com/xhd2015/lifelog-private/ai-critic/server/tool_resolve"

// IsCommandAvailable checks if a command is available on PATH
func IsCommandAvailable(name string) bool {
	return tool_resolve.IsAvailable(name)
}
