package lib

import (
	"github.com/xhd2015/ai-critic/server/agents/opencode_serve_children"
)

// CollectOpencodeServePIDs returns PIDs from the children registry and listeners on extraPorts.
func CollectOpencodeServePIDs(configHome string, extraPorts ...int) ([]int, error) {
	return opencode_serve_children.CollectPIDs(configHome, extraPorts...)
}

// KillOpencodeServePIDs terminates verified opencode serve processes.
func KillOpencodeServePIDs(configHome string, pids []int) (skipped []int, killed []int, err error) {
	return opencode_serve_children.KillPIDs(configHome, pids)
}

// CleanupOpencodeServe collects all opencode serve PIDs, kills them, and clears registries.
func CleanupOpencodeServe(configHome string, extraPorts ...int) error {
	return opencode_serve_children.CleanupAll(configHome, extraPorts...)
}