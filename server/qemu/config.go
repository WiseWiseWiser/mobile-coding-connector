package qemu

import (
	"os"
	"sync"

	"github.com/xhd2015/ai-critic/server/config"
	shared "github.com/xhd2015/dot-pkgs/go-pkgs/qemu"
)

var (
	configFileMu   sync.RWMutex
	configFilePath = config.QemuFile
)

// SetConfigFile overrides the qemu.json path (tests).
func SetConfigFile(path string) {
	configFileMu.Lock()
	defer configFileMu.Unlock()
	configFilePath = path
}

func getConfigFile() string {
	configFileMu.RLock()
	defer configFileMu.RUnlock()
	return configFilePath
}

// LoadConfig reads qemu.json. Missing file yields Enabled=false.
func LoadConfig() (shared.FileConfig, error) {
	cfg, err := shared.LoadFileConfig(getConfigFile())
	if err != nil {
		if os.IsNotExist(err) {
			return shared.FileConfig{}, nil
		}
		return shared.FileConfig{}, err
	}
	return cfg, nil
}

// SaveConfig writes qemu.json.
func SaveConfig(cfg shared.FileConfig) error {
	return shared.SaveFileConfig(getConfigFile(), cfg)
}

// Enabled reports whether qemu guest cloudflared is enabled.
// Missing or unreadable config yields false.
func Enabled() bool {
	return shared.IsEnabled(getConfigFile())
}

// DefaultManager returns a Manager for the server host.
// On the remote ai-critic server this is WorkDevConfig (/root/qemu-guest);
// local-agent CLI uses AiCriticConfig (~/.ai-critic/qemu) separately.
func DefaultManager() *shared.Manager {
	return &shared.Manager{
		Cfg:  shared.WorkDevConfig(),
		Host: shared.LocalHost{},
	}
}

// ManagerWithOptions returns DefaultManager with dry-run and optional writers.
func ManagerWithOptions(dryRun bool, stdout, stderr *os.File) *shared.Manager {
	m := DefaultManager()
	m.DryRun = dryRun
	if stdout != nil {
		m.Stdout = stdout
	}
	if stderr != nil {
		m.Stderr = stderr
	}
	return m
}
