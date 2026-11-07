//go:build !darwin && !linux

package diskmon

import "os"

// hardlinkKey is a no-op where the platform stat layout is unknown; the walk
// still works, it just cannot dedupe hardlinks.
func hardlinkKey(os.FileInfo) (dev, ino uint64, ok bool) {
	return 0, 0, false
}
