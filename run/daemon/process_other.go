//go:build !linux

package daemon

// IsProcessStopped is only implemented on Linux (/proc); always false elsewhere.
func IsProcessStopped(pid int) bool {
	return false
}