//go:build !darwin && !linux

package diskmon

// platformReadMounts has no implementation on this platform, so the client
// hides the disk card instead of showing invented numbers.
func platformReadMounts() ([]mountEntry, error) {
	return nil, nil
}
