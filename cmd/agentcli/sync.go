package agentcli

import (
	"os"
	"path/filepath"

	"github.com/xhd2015/ai-critic/cmd/agentcli/synccmd"
	"github.com/xhd2015/ai-critic/cmd/agentcli/testhooks"
)

// runSync dispatches remote-agent sync with production dirs under ~/.ai-critic
// and ~/.unison (injectable home via testhooks).
func runSync(args []string) error {
	home, err := testhooks.UserHomeDir()
	if err != nil {
		return err
	}
	return synccmd.RunCLI(args, synccmd.CLIOpts{
		StoreDir:     filepath.Join(home, ".ai-critic", "sync"),
		UnisonDir:    filepath.Join(home, ".unison"),
		SSHConfigDir: filepath.Join(home, ".ai-critic", "ssh"),
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
	})
}
