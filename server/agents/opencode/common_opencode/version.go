package common_opencode

import (
	"os/exec"
	"strings"
)

// GetVersion returns the opencode version by running `<binary> --version`.
// Returns empty string if the binary is not found or the command fails.
func GetVersion(binaryPath string) string {
	out, err := exec.Command(binaryPath, "--version").CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
