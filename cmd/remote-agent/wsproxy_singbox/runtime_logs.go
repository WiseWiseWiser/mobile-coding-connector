package wsproxy_singbox

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

const (
	xraySidecarLogFile = "xray-sidecar.log"
	singBoxLogFile     = "sing-box.log"
)

// agentCacheDir returns ~/Library/Caches/remote-agent (or the invoking user's
// cache when remote-agent is started via sudo). Logs live here so a prior root
// run cannot leave cwd log files that block non-sudo use.
func agentCacheDir() (string, error) {
	base, err := processCacheBaseDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "remote-agent")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

func processCacheBaseDir() (string, error) {
	if os.Geteuid() == 0 {
		if name := os.Getenv("SUDO_USER"); name != "" && name != "root" {
			u, err := user.Lookup(name)
			if err == nil {
				return homeCacheDir(u.HomeDir), nil
			}
		}
	}
	return os.UserCacheDir()
}

func homeCacheDir(home string) string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Caches")
	}
	if dir := os.Getenv("XDG_CACHE_HOME"); dir != "" {
		return dir
	}
	return filepath.Join(home, ".cache")
}

func xraySidecarLogPath() string {
	dir, err := agentCacheDir()
	if err != nil {
		return xraySidecarLogFile
	}
	return filepath.Join(dir, xraySidecarLogFile)
}

func singBoxLogPath() string {
	dir, err := agentCacheDir()
	if err != nil {
		return singBoxLogFile
	}
	return filepath.Join(dir, singBoxLogFile)
}