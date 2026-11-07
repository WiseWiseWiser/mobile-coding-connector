//go:build darwin

package diskmon

import (
	"os"
	"syscall"
)

// hardlinkKey identifies a file with more than one link so a volume walk counts
// its bytes once. Regular one-link files skip the map entirely.
func hardlinkKey(fi os.FileInfo) (dev, ino uint64, ok bool) {
	st, isStat := fi.Sys().(*syscall.Stat_t)
	if !isStat || st.Nlink <= 1 {
		return 0, 0, false
	}
	return uint64(uint32(st.Dev)), st.Ino, true
}
