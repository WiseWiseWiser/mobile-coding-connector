//go:build linux

package diskmon

import (
	"os"
	"strings"
	"syscall"
)

const stReadOnly = 0x1

// platformReadMounts parses /proc/self/mounts and stats every mount point.
// Mounts that cannot be stat'ed are dropped: capacity is the whole point.
func platformReadMounts() ([]mountEntry, error) {
	data, err := os.ReadFile("/proc/self/mounts")
	if err != nil {
		return nil, err
	}
	out := make([]mountEntry, 0, 64)
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		mount := unescapeMount(fields[1])
		var st syscall.Statfs_t
		if err := syscall.Statfs(mount, &st); err != nil {
			continue
		}
		out = append(out, mountEntry{
			Mount:    mount,
			Device:   unescapeMount(fields[0]),
			FSType:   fields[2],
			Blocks:   st.Blocks,
			Bfree:    st.Bfree,
			Bavail:   st.Bavail,
			Bsize:    uint64(st.Bsize),
			ReadOnly: st.Flags&stReadOnly != 0,
		})
	}
	return out, nil
}
