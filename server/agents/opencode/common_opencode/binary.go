package common_opencode

import (
	"os"
	"os/exec"
)

// IsBinaryAvailable reports whether the opencode executable can be launched.
func IsBinaryAvailable(customPath string) bool {
	if customPath != "" {
		info, err := os.Stat(customPath)
		return err == nil && !info.IsDir()
	}
	_, err := exec.LookPath("opencode")
	return err == nil
}