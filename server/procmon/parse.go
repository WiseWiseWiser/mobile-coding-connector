package procmon

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// parsePSTable parses `ps -axo pid,ppid,uid,user,state,rss,etime,time,command`
// (Linux may use cputime in the TIME column). Header row is skipped.
func parsePSTable(output string) ([]rawProc, error) {
	var out []rawProc
	for i, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}
		if i == 0 && (fields[0] == "PID" || fields[0] == "USER") {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		ppid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		uid, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}
		rss, err := strconv.ParseUint(fields[5], 10, 64)
		if err != nil {
			continue
		}
		elapsed, err := parsePSClock(fields[6])
		if err != nil {
			elapsed = 0
		}
		cpuTime, err := parsePSClock(fields[7])
		if err != nil {
			cpuTime = 0
		}
		out = append(out, rawProc{
			PID:     pid,
			PPID:    ppid,
			UID:     uid,
			User:    fields[3],
			State:   fields[4],
			RSSKiB:  rss,
			Elapsed: elapsed,
			CPUTime: cpuTime,
			Cmd:     strings.Join(fields[8:], " "),
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no processes parsed from ps output")
	}
	return out, nil
}

// parsePSClock parses ps etime/cputime: [[dd-]hh:]mm:ss[.frac] or mm:ss.frac.
func parsePSClock(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" {
		return 0, fmt.Errorf("empty clock")
	}
	var days int
	if i := strings.IndexByte(s, '-'); i >= 0 {
		d, err := strconv.Atoi(s[:i])
		if err != nil {
			return 0, err
		}
		days = d
		s = s[i+1:]
	}
	parts := strings.Split(s, ":")
	var hours, minutes int
	var seconds float64
	var err error
	switch len(parts) {
	case 3:
		hours, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, err
		}
		minutes, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, err
		}
		seconds, err = strconv.ParseFloat(parts[2], 64)
		if err != nil {
			return 0, err
		}
	case 2:
		minutes, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, err
		}
		seconds, err = strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return 0, err
		}
	default:
		return 0, fmt.Errorf("bad clock %q", s)
	}
	total := float64(days)*86400 + float64(hours)*3600 + float64(minutes)*60 + seconds
	return time.Duration(total * float64(time.Second)), nil
}

func displayName(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return "?"
	}
	if i := strings.Index(cmd, ".app"); i >= 0 {
		start := strings.LastIndex(cmd[:i], "/")
		name := cmd[start+1 : i]
		if name != "" {
			return name
		}
	}
	exe := argv0(cmd)
	base := filepath.Base(exe)
	if base == "" || base == "." || base == string(filepath.Separator) {
		return exe
	}
	return base
}

// argv0 is the executable path. Absolute paths may contain spaces
// ("/Library/Application Support/..."); flags start at " --" or " -".
func argv0(cmd string) string {
	if cmd == "" {
		return cmd
	}
	if cmd[0] == '/' || strings.HasPrefix(cmd, "./") || strings.HasPrefix(cmd, "../") {
		if i := strings.Index(cmd, " --"); i >= 0 {
			return strings.TrimSpace(cmd[:i])
		}
		if i := strings.Index(cmd, " -"); i >= 0 {
			return strings.TrimSpace(cmd[:i])
		}
		return cmd
	}
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return cmd
	}
	return fields[0]
}

func parseLoadAvg(s string) (Load, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "{")
	s = strings.TrimSuffix(s, "}")
	fields := strings.Fields(s)
	if len(fields) < 3 {
		return Load{}, fmt.Errorf("loadavg: %q", s)
	}
	m1, err := strconv.ParseFloat(strings.TrimSuffix(fields[0], ","), 64)
	if err != nil {
		return Load{}, err
	}
	m5, err := strconv.ParseFloat(strings.TrimSuffix(fields[1], ","), 64)
	if err != nil {
		return Load{}, err
	}
	m15, err := strconv.ParseFloat(strings.TrimSuffix(fields[2], ","), 64)
	if err != nil {
		return Load{}, err
	}
	return Load{M1: m1, M5: m5, M15: m15}, nil
}

func parseSwapUsage(s string) (total, used, free uint64) {
	// vm.swapusage: total = 8192.00M  used = 6526.31M  free = 1665.69M  (encrypted)
	lower := strings.ToLower(s)
	total = parseSwapField(lower, "total")
	used = parseSwapField(lower, "used")
	free = parseSwapField(lower, "free")
	return total, used, free
}

func parseSwapField(s, key string) uint64 {
	idx := strings.Index(s, key)
	if idx < 0 {
		return 0
	}
	rest := s[idx+len(key):]
	eq := strings.IndexByte(rest, '=')
	if eq < 0 {
		return 0
	}
	rest = strings.TrimSpace(rest[eq+1:])
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return 0
	}
	return parseSizeToken(fields[0])
}

func parseSizeToken(tok string) uint64 {
	tok = strings.TrimSpace(tok)
	if tok == "" {
		return 0
	}
	mult := uint64(1)
	switch {
	case strings.HasSuffix(tok, "G") || strings.HasSuffix(tok, "g"):
		mult = 1 << 30
		tok = tok[:len(tok)-1]
	case strings.HasSuffix(tok, "M") || strings.HasSuffix(tok, "m"):
		mult = 1 << 20
		tok = tok[:len(tok)-1]
	case strings.HasSuffix(tok, "K") || strings.HasSuffix(tok, "k"):
		mult = 1 << 10
		tok = tok[:len(tok)-1]
	}
	f, err := strconv.ParseFloat(tok, 64)
	if err != nil {
		return 0
	}
	return uint64(f * float64(mult))
}

func parseDarwinVmStat(data string, total uint64) Memory {
	pageSize := uint64(4096)
	var free, active, inactive, speculative, wired, compressor uint64
	for _, line := range strings.Split(data, "\n") {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "page size of") {
			fields := strings.Fields(line)
			for i, f := range fields {
				if f == "of" && i+1 < len(fields) {
					n, err := strconv.ParseUint(strings.TrimSpace(fields[i+1]), 10, 64)
					if err == nil && n > 0 {
						pageSize = n
					}
					break
				}
			}
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		raw := strings.TrimSuffix(fields[len(fields)-1], ".")
		n, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			continue
		}
		switch {
		case strings.HasPrefix(line, "Pages free:"):
			free = n
		case strings.HasPrefix(line, "Pages active:"):
			active = n
		case strings.HasPrefix(line, "Pages inactive:"):
			inactive = n
		case strings.HasPrefix(line, "Pages speculative:"):
			speculative = n
		case strings.HasPrefix(line, "Pages wired down:"):
			wired = n
		case strings.HasPrefix(line, "Pages occupied by compressor:"):
			compressor = n
		}
	}
	available := (free + inactive + speculative) * pageSize
	used := (active + wired + compressor) * pageSize
	if total > 0 && available < total {
		used = total - available
	}
	var usedPercent float64
	if total > 0 {
		usedPercent = float64(used) / float64(total) * 100
	}
	return Memory{
		Total:           total,
		Used:            used,
		Free:            free * pageSize,
		UsedPercent:     usedPercent,
		CompressorBytes: compressor * pageSize,
	}
}

func parseLinuxMeminfo(data string) Memory {
	var memTotal, memFree, memAvailable, swapTotal, swapFree uint64
	for _, line := range strings.Split(data, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		value *= 1024
		switch fields[0] {
		case "MemTotal:":
			memTotal = value
		case "MemFree:":
			memFree = value
		case "MemAvailable:":
			memAvailable = value
		case "SwapTotal:":
			swapTotal = value
		case "SwapFree:":
			swapFree = value
		}
	}
	if memAvailable == 0 {
		memAvailable = memFree
	}
	var used uint64
	if memTotal > memAvailable {
		used = memTotal - memAvailable
	}
	var usedPercent float64
	if memTotal > 0 {
		usedPercent = float64(used) / float64(memTotal) * 100
	}
	var swapUsed uint64
	if swapTotal > swapFree {
		swapUsed = swapTotal - swapFree
	}
	var swapPct float64
	if swapTotal > 0 {
		swapPct = float64(swapUsed) / float64(swapTotal) * 100
	}
	return Memory{
		Total:           memTotal,
		Used:            used,
		Free:            memFree,
		UsedPercent:     usedPercent,
		SwapTotal:       swapTotal,
		SwapUsed:        swapUsed,
		SwapFree:        swapFree,
		SwapUsedPercent: swapPct,
	}
}
