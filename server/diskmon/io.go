package diskmon

import (
	"time"

	serverhostmon "github.com/xhd2015/ai-critic/server/hostmon"
)

// latestHostIO reuses hostmon's newest sample instead of re-reading
// /proc/diskstats. macOS has no persisted sample, so the field stays absent.
func latestHostIO() (IO, bool) {
	sample, ok := serverhostmon.Latest()
	if !ok || sample.T == "" {
		return IO{}, false
	}
	return IO{
		Device:      sample.Disk.Dev,
		ReadBps:     sample.Disk.RBps,
		WriteBps:    sample.Disk.WBps,
		ReadIOPS:    sample.Disk.RIops,
		WriteIOPS:   sample.Disk.WIops,
		IntervalSec: int(serverhostmon.Interval / time.Second),
		SampledAt:   sample.T,
	}, true
}
