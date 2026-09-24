//go:build unix

package fileupload

import (
	"os"
	"syscall"
)

// preserveOwner copies the uid/gid of the original file onto path. Best effort:
// chown fails with EPERM for unprivileged users and the resulting file is
// written as the server process owner, which is the common case anyway.
func preserveOwner(path string, info os.FileInfo) {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return
	}
	_ = os.Chown(path, int(st.Uid), int(st.Gid))
}
