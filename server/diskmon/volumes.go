package diskmon

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// dataMount holds the writable half of a modern macOS boot volume. "/" is a
// read-only sealed snapshot of the same APFS container.
const dataMount = "/System/Volumes/Data"

// mountEntry is one mount table row plus its capacity counters.
type mountEntry struct {
	Mount    string
	Device   string
	FSType   string
	Blocks   uint64
	Bfree    uint64
	Bavail   uint64
	Bsize    uint64
	ReadOnly bool
}

// pseudoFS never represents user-visible storage.
var pseudoFS = map[string]bool{
	"devfs":       true,
	"autofs":      true,
	"tmpfs":       true,
	"devtmpfs":    true,
	"proc":        true,
	"procfs":      true,
	"sysfs":       true,
	"cgroup":      true,
	"cgroup2":     true,
	"pstore":      true,
	"securityfs":  true,
	"debugfs":     true,
	"tracefs":     true,
	"bpf":         true,
	"configfs":    true,
	"fusectl":     true,
	"mqueue":      true,
	"hugetlbfs":   true,
	"ramfs":       true,
	"nsfs":        true,
	"binfmt_misc": true,
	"rpc_pipefs":  true,
	"overlay":     true,
	"squashfs":    true,
	"efivarfs":    true,
	"selinuxfs":   true,
	"fuse.portal": true,
}

// skipPrefixes are mount points that exist but are not storage the user cares
// about: snap packages, container overlays and macOS support volumes.
var skipPrefixes = []string{
	"/snap/",
	"/var/lib/docker/",
	"/var/lib/kubelet/",
	"/run/user/",
	"/System/Volumes/VM",
	"/System/Volumes/Preboot",
	"/System/Volumes/Update",
	"/System/Volumes/xarts",
	"/System/Volumes/iSCPreboot",
	"/System/Volumes/Hardware",
	"/private/var/vm",
	// Xcode simulator runtime images: read-only, tiny, many of them.
	"/Library/Developer/CoreSimulator/",
}

var skipMounts = map[string]bool{
	"/boot/efi":      true,
	"/boot/firmware": true,
	"/dev":           true,
}

// buildSnapshot filters a raw mount table into the API snapshot.
func buildSnapshot(entries []mountEntry, now time.Time) *Snapshot {
	snap := &Snapshot{
		SampledAt: now.UTC().Format(time.RFC3339),
		Volumes:   []Volume{},
	}

	hasData := false
	for _, e := range entries {
		if cleanMount(e.Mount) == dataMount {
			hasData = true
			break
		}
	}

	seenDevice := map[string]bool{}
	unreadable := 0
	for _, e := range entries {
		mount := cleanMount(e.Mount)
		if skipEntry(mount, e) {
			continue
		}
		if hasData && mount == "/" && e.ReadOnly {
			snap.Merged = append(snap.Merged, MergedMount{
				Mount:  mount,
				Device: e.Device,
				FSType: e.FSType,
				Reason: "same APFS container as " + dataMount + " — counted once",
			})
			continue
		}
		if e.Blocks == 0 || e.Bsize == 0 {
			unreadable++
			continue
		}
		if seenDevice[e.Device] {
			continue
		}
		seenDevice[e.Device] = true

		total := e.Blocks * e.Bsize
		free := e.Bavail * e.Bsize
		if free > total {
			free = total
		}
		used := total
		if reserved := e.Bfree * e.Bsize; reserved < total {
			used = total - reserved
		}
		snap.Volumes = append(snap.Volumes, Volume{
			Mount:       mount,
			Device:      e.Device,
			FSType:      e.FSType,
			TotalBytes:  int64(total),
			UsedBytes:   int64(used),
			FreeBytes:   int64(free),
			UsedPercent: percentOf(used, total),
			ReadOnly:    e.ReadOnly,
		})
	}

	if primary := primaryIndex(snap.Volumes); primary >= 0 {
		snap.Volumes[primary].Primary = true
		snap.PrimaryMount = snap.Volumes[primary].Mount
	}
	sort.SliceStable(snap.Volumes, func(i, j int) bool {
		if snap.Volumes[i].Primary != snap.Volumes[j].Primary {
			return snap.Volumes[i].Primary
		}
		return snap.Volumes[i].Mount < snap.Volumes[j].Mount
	})
	sort.SliceStable(snap.Merged, func(i, j int) bool {
		return snap.Merged[i].Mount < snap.Merged[j].Mount
	})
	if unreadable > 0 && len(snap.Volumes) > 0 {
		snap.Warning = fmt.Sprintf("%d mount(s) could not be read", unreadable)
	}
	return snap
}

// unescapeMount decodes the octal escapes /proc/self/mounts uses.
func unescapeMount(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	replacer := strings.NewReplacer(
		`\040`, " ",
		`\011`, "\t",
		`\012`, "\n",
		`\134`, `\`,
	)
	return replacer.Replace(s)
}

func cleanMount(mount string) string {
	trimmed := strings.TrimRight(mount, "/")
	if trimmed == "" {
		return "/"
	}
	return trimmed
}

func skipEntry(mount string, e mountEntry) bool {
	if pseudoFS[strings.ToLower(e.FSType)] {
		return true
	}
	if strings.HasPrefix(e.Device, "map ") {
		return true
	}
	if skipMounts[mount] {
		return true
	}
	for _, prefix := range skipPrefixes {
		if mount == strings.TrimSuffix(prefix, "/") || strings.HasPrefix(mount, prefix) {
			return true
		}
	}
	return false
}

// primaryIndex prefers the volume that actually holds user data.
func primaryIndex(volumes []Volume) int {
	for i := range volumes {
		if volumes[i].Mount == dataMount {
			return i
		}
	}
	for i := range volumes {
		if volumes[i].Mount == "/" {
			return i
		}
	}
	if len(volumes) > 0 {
		return 0
	}
	return -1
}

func percentOf(used, total uint64) float64 {
	if total == 0 {
		return 0
	}
	pct := float64(used) / float64(total) * 100
	return math.Round(pct*10) / 10
}
