package lib

import "github.com/xhd2015/lifelog-private/ai-critic/server/config"

const (
	// BinaryName is the name of the server binary used in releases and installs.
	BinaryName = "ai-critic-server"

	// DefaultServerPort is the default port for the Go backend server.
	// Re-exported from config for backward compatibility.
	DefaultServerPort = config.DefaultServerPort

	// ViteDevPort is the port where Vite dev server runs (only used by scripts).
	ViteDevPort = 5173

	// QuickTestPort is the default port for quick-test mode.
	QuickTestPort = 37651
)
