package wsproxy_singbox

import (
	"fmt"

	"github.com/xhd2015/dot-pkgs/go-pkgs/sudosetup"
)

// defaultEnsureSudoSetup remains the TestHooks default so InstallTestHooks /
// bridgeHooksToSingboxtun can audit sudo setup. singboxtun.RunTun invokes it
// via the bridged hook (CacheDirName/SudoersName match tunCacheDirName).
func defaultEnsureSudoSetup(singBoxPath string, noSetup bool) error {
	if noSetup {
		return nil
	}
	mgr := &sudosetup.Manager{
		Config: sudosetup.Config{
			CacheDirName: tunCacheDirName,
			SudoersName:  tunSudoersName,
		},
		Rule: sudosetup.Rule{
			Command:     singBoxPath,
			ArgsPattern: "run -c *",
		},
	}
	if installed, _ := mgr.IsInstalled(); installed {
		return nil
	}
	if err := mgr.EnsureInstalled(); err != nil {
		return fmt.Errorf("sudo NOPASSWD setup: %w", err)
	}
	return nil
}