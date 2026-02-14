package server

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type ServerStatus struct {
	Memory MemoryStatus    `json:"memory"`
	Disk   []DiskStatus    `json:"disk"`
	CPU    CPUStatus       `json:"cpu"`
	OSInfo OSInfo          `json:"os_info"`
	TopCPU []ProcessStatus `json:"top_cpu"`
	TopMem []ProcessStatus `json:"top_mem"`
}

type MemoryStatus struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
}

type DiskStatus struct {
	Filesystem string  `json:"filesystem"`
	Size       uint64  `json:"size"`
	Used       uint64  `json:"used"`
	Available  uint64  `json:"available"`
	UsePercent float64 `json:"use_percent"`
	MountPoint string  `json:"mount_point"`
}

type CPUStatus struct {
	NumCPU      int     `json:"num_cpu"`
	UsedPercent float64 `json:"used_percent"`
}

type OSInfo struct {
	OS      string `json:"os"`
	Arch    string `json:"arch"`
	Kernel  string `json:"kernel"`
	Version string `json:"version"`
}

type ProcessStatus struct {
	PID     int    `json:"pid"`
	Name    string `json:"name"`
	CPU     string `json:"cpu"`
	Mem     string `json:"mem"`
	Command string `json:"command"`
}

func RegisterServerStatusAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/server/status", handleServerStatus)
}

func handleServerStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status, err := getServerStatus()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func getServerStatus() (*ServerStatus, error) {
	mem, err := getMemoryStatus()
	if err != nil {
		return nil, err
	}

	disk, err := getDiskStatus()
	if err != nil {
		return nil, err
	}

	cpu, err := getCPUStatus()
	if err != nil {
		return nil, err
	}

	osInfo, err := getOSInfo()
	if err != nil {
		return nil, err
	}

	topCPU, err := getTopProcessesByCPU(3)
	if err != nil {
		return nil, err
	}

	topMem, err := getTopProcessesByMem(3)
	if err != nil {
		return nil, err
	}

	return &ServerStatus{
		Memory: mem,
		Disk:   disk,
		CPU:    cpu,
		OSInfo: osInfo,
		TopCPU: topCPU,
		TopMem: topMem,
	}, nil
}

func getMemoryStatus() (MemoryStatus, error) {
	var memStatus MemoryStatus

	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return memStatus, err
	}

	var memTotal, memFree, memAvailable uint64
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
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

	used := memTotal - memAvailable
	usedPercent := float64(used) / float64(memTotal) * 100

	return MemoryStatus{
		Total:       memTotal,
		Used:        used,
		Free:        memFree,
		UsedPercent: usedPercent,
	}, nil
}

func getDiskStatus() ([]DiskStatus, error) {
	var disks []DiskStatus

	cmd := exec.Command("df", "-B1", "--output=source,size,used,avail,target")
	output, err := cmd.Output()
	if err != nil {
		return disks, err
	}

	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		size, _ := strconv.ParseUint(fields[1], 10, 64)
		used, _ := strconv.ParseUint(fields[2], 10, 64)
		avail, _ := strconv.ParseUint(fields[3], 10, 64)

		var usePercent float64
		if size > 0 {
			usePercent = float64(used) / float64(size) * 100
		}

		disks = append(disks, DiskStatus{
			Filesystem: fields[0],
			Size:       size,
			Used:       used,
			Available:  avail,
			UsePercent: usePercent,
			MountPoint: fields[4],
		})
	}

	return disks, nil
}

func getCPUStatus() (CPUStatus, error) {
	var cpuStatus CPUStatus
	cpuStatus.NumCPU = runtime.NumCPU()

	cmd := exec.Command("top", "-bn1")
	output, err := cmd.Output()
	if err != nil {
		return cpuStatus, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Cpu(s)") || strings.HasPrefix(line, "%Cpu(s)") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if strings.Contains(field, "id") {
					idleStr := fields[i-1]
					idle := parseFloat(strings.TrimSuffix(idleStr, ","))
					cpuStatus.UsedPercent = 100 - idle
					break
				}
			}
			break
		}
	}

	return cpuStatus, nil
}

func getOSInfo() (OSInfo, error) {
	var osInfo OSInfo

	data, err := os.ReadFile("/etc/os-release")
	if err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				osInfo.OS = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
				break
			}
		}
	}

	if osInfo.OS == "" {
		cmd := exec.Command("uname", "-o")
		output, _ := cmd.Output()
		osInfo.OS = strings.TrimSpace(string(output))
	}

	cmd := exec.Command("uname", "-r")
	output, _ := cmd.Output()
	osInfo.Kernel = strings.TrimSpace(string(output))

	cmd = exec.Command("uname", "-m")
	output, _ = cmd.Output()
	osInfo.Arch = strings.TrimSpace(string(output))

	osInfo.Version = runtime.Version()

	return osInfo, nil
}

func getTopProcessesByCPU(n int) ([]ProcessStatus, error) {
	return getTopProcesses(n, "cpu")
}

func getTopProcessesByMem(n int) ([]ProcessStatus, error) {
	return getTopProcesses(n, "mem")
}

func getTopProcesses(n int, sortBy string) ([]ProcessStatus, error) {
	var processes []ProcessStatus

	cmd := exec.Command("ps", "aux", "--no-headers")
	output, err := cmd.Output()
	if err != nil {
		return processes, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	type proc struct {
		pid  int
		cpu  float64
		mem  float64
		name string
		cmd  string
	}

	var procs []proc
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 11 {
			continue
		}

		pid, _ := strconv.Atoi(fields[1])
		cpu, _ := strconv.ParseFloat(fields[2], 64)
		mem, _ := strconv.ParseFloat(fields[3], 64)
		name := fields[10]
		cmdline := strings.Join(fields[10:], " ")

		procs = append(procs, proc{pid: pid, cpu: cpu, mem: mem, name: name, cmd: cmdline})
	}

	sorted := make([]proc, len(procs))
	copy(sorted, procs)

	if sortBy == "cpu" {
		for i := 0; i < len(sorted)-1; i++ {
			for j := i + 1; j < len(sorted); j++ {
				if sorted[j].cpu > sorted[i].cpu {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}
	} else {
		for i := 0; i < len(sorted)-1; i++ {
			for j := i + 1; j < len(sorted); j++ {
				if sorted[j].mem > sorted[i].mem {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}
	}

	count := n
	if count > len(sorted) {
		count = len(sorted)
	}

	for i := 0; i < count; i++ {
		processes = append(processes, ProcessStatus{
			PID:     sorted[i].pid,
			Name:    sorted[i].name,
			CPU:     strconv.FormatFloat(sorted[i].cpu, 'f', 1, 64) + "%",
			Mem:     strconv.FormatFloat(sorted[i].mem, 'f', 1, 64) + "%",
			Command: sorted[i].cmd,
		})
	}

	return processes, nil
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
