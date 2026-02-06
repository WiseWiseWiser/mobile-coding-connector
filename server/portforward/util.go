package portforward

import "os/exec"

// IsCommandAvailable checks if a command is available on PATH
func IsCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
