//go:build linux

package diskmon

import (
	"os"
	"syscall"
)

func hardlinkKey(fi os.FileInfo) (dev, ino uint64, ok bool) {
	st, isStat := fi.Sys().(*syscall.Stat_t)
	if !isStat || st.Nlink <= 1 {
		return 0, 0, false
	}
	return st.Dev, st.Ino, true
}
