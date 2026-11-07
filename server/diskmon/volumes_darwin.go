//go:build darwin

package diskmon

import "syscall"

const (
	mntNowait   = 2
	mntReadOnly = 0x00000001
)

// platformReadMounts reads the mount table through Getfsstat, which already
// carries per-mount capacity counters.
func platformReadMounts() ([]mountEntry, error) {
	count, err := syscall.Getfsstat(nil, mntNowait)
	if err != nil {
		return nil, err
	}
	if count <= 0 {
		return nil, nil
	}
	buf := make([]syscall.Statfs_t, count+count/4+8)
	count, err = syscall.Getfsstat(buf, mntNowait)
	if err != nil {
		return nil, err
	}
	if count > len(buf) {
		count = len(buf)
	}
	out := make([]mountEntry, 0, count)
	for i := 0; i < count; i++ {
		st := buf[i]
		out = append(out, mountEntry{
			Mount:    cstr(st.Mntonname[:]),
			Device:   cstr(st.Mntfromname[:]),
			FSType:   cstr(st.Fstypename[:]),
			Blocks:   st.Blocks,
			Bfree:    st.Bfree,
			Bavail:   st.Bavail,
			Bsize:    uint64(st.Bsize),
			ReadOnly: st.Flags&mntReadOnly != 0,
		})
	}
	return out, nil
}

// cstr converts a NUL-terminated int8 array (the darwin statfs layout).
func cstr(buf []int8) string {
	out := make([]byte, 0, len(buf))
	for _, c := range buf {
		if c == 0 {
			break
		}
		out = append(out, byte(c))
	}
	return string(out)
}
