package procmon

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

func runPS() (string, error) {
	cmd := exec.Command("ps", "-axo", "pid,ppid,uid,user,state,rss,etime,time,command")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func readLoad() (Load, error) {
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("sysctl", "-n", "vm.loadavg").Output()
		if err != nil {
			return Load{}, err
		}
		return parseLoadAvg(string(out))
	}
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return Load{}, err
	}
	return parseLoadAvg(string(data))
}

func readMemory() Memory {
	if runtime.GOOS == "darwin" {
		return readDarwinMemory()
	}
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return Memory{}
	}
	return parseLinuxMeminfo(string(data))
}

func readDarwinMemory() Memory {
	totalOut, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return Memory{}
	}
	total, err := strconv.ParseUint(strings.TrimSpace(string(totalOut)), 10, 64)
	if err != nil {
		return Memory{}
	}
	vmOut, err := exec.Command("vm_stat").Output()
	if err != nil {
		return Memory{Total: total}
	}
	mem := parseDarwinVmStat(string(vmOut), total)
	swapOut, err := exec.Command("sysctl", "-n", "vm.swapusage").Output()
	if err == nil {
		totalSw, usedSw, freeSw := parseSwapUsage(string(swapOut))
		mem.SwapTotal = totalSw
		mem.SwapUsed = usedSw
		mem.SwapFree = freeSw
		if totalSw > 0 {
			mem.SwapUsedPercent = float64(usedSw) / float64(totalSw) * 100
		}
	}
	return mem
}
