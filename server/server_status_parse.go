package server

import (
	"strconv"
	"strings"
)

func parseLinuxMeminfo(data string) (MemoryStatus, error) {
	var memTotal, memFree, memAvailable uint64
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
		}
	}
	if memAvailable == 0 {
		memAvailable = memFree
	}
	var used uint64
	var usedPercent float64
	if memTotal > memAvailable {
		used = memTotal - memAvailable
	}
	if memTotal > 0 {
		usedPercent = float64(used) / float64(memTotal) * 100
	}
	return MemoryStatus{
		Total:       memTotal,
		Used:        used,
		Free:        memFree,
		UsedPercent: usedPercent,
	}, nil
}

func parseDarwinVmStat(data string, total uint64) (MemoryStatus, error) {
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
	return MemoryStatus{
		Total:       total,
		Used:        used,
		Free:        free * pageSize,
		UsedPercent: usedPercent,
	}, nil
}

func parseGNUDfOutput(output string) []DiskStatus {
	var disks []DiskStatus
	for i, line := range strings.Split(output, "\n") {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		disks = append(disks, diskFromSizeFields(fields[0], fields[1], fields[2], fields[3], fields[4], 1))
	}
	return disks
}

func parsePOSIXDfKP(output string, blockSize uint64) []DiskStatus {
	var disks []DiskStatus
	for i, line := range strings.Split(output, "\n") {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		mount := fields[len(fields)-1]
		disks = append(disks, diskFromSizeFields(fields[0], fields[1], fields[2], fields[3], mount, blockSize))
	}
	return disks
}

func diskFromSizeFields(fs, sizeStr, usedStr, availStr, mount string, blockSize uint64) DiskStatus {
	size, _ := strconv.ParseUint(sizeStr, 10, 64)
	used, _ := strconv.ParseUint(usedStr, 10, 64)
	avail, _ := strconv.ParseUint(availStr, 10, 64)
	if blockSize > 1 {
		size *= blockSize
		used *= blockSize
		avail *= blockSize
	}
	var usePercent float64
	if size > 0 {
		usePercent = float64(used) / float64(size) * 100
	}
	return DiskStatus{
		Filesystem: fs,
		Size:       size,
		Used:       used,
		Available:  avail,
		UsePercent: usePercent,
		MountPoint: mount,
	}
}

func parseLinuxTopCPU(output string) float64 {
	for _, line := range strings.Split(output, "\n") {
		if !strings.Contains(line, "Cpu(s)") && !strings.HasPrefix(line, "%Cpu(s)") {
			continue
		}
		fields := strings.Fields(line)
		for i, field := range fields {
			if strings.Contains(field, "id") && i > 0 {
				idleStr := strings.TrimSuffix(fields[i-1], ",")
				return 100 - parseFloat(idleStr)
			}
		}
	}
	return 0
}

func parseDarwinTopCPU(output string) float64 {
	for _, line := range strings.Split(output, "\n") {
		if !strings.Contains(line, "CPU usage:") {
			continue
		}
		fields := strings.Fields(line)
		for i, field := range fields {
			if strings.Contains(field, "idle") && i > 0 {
				idleStr := strings.TrimSuffix(fields[i-1], "%")
				idleStr = strings.TrimSuffix(idleStr, ",")
				return 100 - parseFloat(idleStr)
			}
		}
	}
	return 0
}

func parsePrettyName(osRelease string) string {
	for _, line := range strings.Split(osRelease, "\n") {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			return strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
		}
	}
	return ""
}
