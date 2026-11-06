package procmon

import (
	"math"
	"runtime"
	"sort"
	"time"
)

func buildSnapshot(prev, cur []rawProc, dt time.Duration, at time.Time, mem Memory, load Load, selfPID int, username string, ncpu int) *Snapshot {
	if ncpu <= 0 {
		ncpu = runtime.NumCPU()
	}
	if dt <= 0 {
		dt = time.Millisecond
	}
	prevByPID := make(map[int]rawProc, len(prev))
	for _, p := range prev {
		prevByPID[p.PID] = p
	}
	curByPID := make(map[int]rawProc, len(cur))
	for _, p := range cur {
		curByPID[p.PID] = p
	}
	protected := ancestorSet(selfPID, curByPID)

	procs := make([]Process, 0, len(cur))
	var cpuSum float64
	for _, p := range cur {
		var cpuPct float64
		if old, ok := prevByPID[p.PID]; ok {
			deltaCPU := p.CPUTime - old.CPUTime
			if deltaCPU < 0 {
				deltaCPU = 0
			}
			cpuPct = (float64(deltaCPU) / float64(dt)) * 100
		}
		cpuPct = round1(cpuPct)
		cpuSum += cpuPct
		reason := protectReason(p, protected)
		item := Process{
			PID:        p.PID,
			PPID:       p.PPID,
			Name:       displayName(p.Cmd),
			Cmd:        p.Cmd,
			User:       p.User,
			State:      p.State,
			CPUPercent: cpuPct,
			RSSBytes:   p.RSSKiB * 1024,
			ElapsedSec: int64(p.Elapsed / time.Second),
			Tags:       tagsFor(p, username),
		}
		if reason != "" {
			item.Protected = reason
		}
		procs = append(procs, item)
	}
	sort.Slice(procs, func(i, j int) bool {
		if procs[i].CPUPercent == procs[j].CPUPercent {
			return procs[i].RSSBytes > procs[j].RSSBytes
		}
		return procs[i].CPUPercent > procs[j].CPUPercent
	})

	used := cpuSum / float64(ncpu)
	if used > 100 {
		used = 100
	}
	if used < 0 {
		used = 0
	}
	return &Snapshot{
		SampledAt:  at.Format(time.RFC3339),
		IntervalMs: int(dt / time.Millisecond),
		CPU: CPU{
			NumCPU:      ncpu,
			UsedPercent: round1(used),
		},
		Load:      load,
		Memory:    mem,
		Counts:    Counts{Processes: len(procs)},
		SelfPID:   selfPID,
		Processes: procs,
	}
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}
