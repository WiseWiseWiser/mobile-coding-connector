package lib

import "github.com/xhd2015/lifelog-private/ai-critic/server/config"

const (
	// DefaultServerPort is the default port for the Go backend server.
	// Re-exported from config for backward compatibility.
	DefaultServerPort = config.DefaultServerPort

	// ViteDevPort is the port where Vite dev server runs (only used by scripts).
	ViteDevPort = 5173
)
