package server

import (
	"encoding/json"
	"fmt"
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
	mux.HandleFunc("/api/server/os-info", handleOSInfo)
}

// handleOSInfo returns uname/os-release only (no df/ps). Used by
// remote-agent install so GOOS/GOARCH probing cannot stall on disk.
func handleOSInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	info, err := getOSInfo()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func handleServerStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(getServerStatus())
}

func getServerStatus() *ServerStatus {
	status := &ServerStatus{}

	mem, err := getMemoryStatus()
	if err != nil {
		fmt.Printf("[server-status] memory: %v\n", err)
	} else {
		status.Memory = mem
	}

	disk, err := getDiskStatus()
	if err != nil {
		fmt.Printf("[server-status] disk: %v\n", err)
	} else {
		status.Disk = disk
	}

	cpu, err := getCPUStatus()
	if err != nil {
		fmt.Printf("[server-status] cpu: %v\n", err)
	} else {
		status.CPU = cpu
	}

	osInfo, err := getOSInfo()
	if err != nil {
		fmt.Printf("[server-status] os: %v\n", err)
	} else {
		status.OSInfo = osInfo
	}

	topCPU, err := getTopProcessesByCPU(3)
	if err != nil {
		fmt.Printf("[server-status] top cpu: %v\n", err)
	} else {
		status.TopCPU = topCPU
	}

	topMem, err := getTopProcessesByMem(3)
	if err != nil {
		fmt.Printf("[server-status] top mem: %v\n", err)
	} else {
		status.TopMem = topMem
	}

	if status.Disk == nil {
		status.Disk = []DiskStatus{}
	}
	if status.TopCPU == nil {
		status.TopCPU = []ProcessStatus{}
	}
	if status.TopMem == nil {
		status.TopMem = []ProcessStatus{}
	}
	return status
}

func getMemoryStatus() (MemoryStatus, error) {
	if runtime.GOOS == "darwin" {
		return getDarwinMemoryStatus()
	}
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return MemoryStatus{}, err
	}
	return parseLinuxMeminfo(string(data))
}

func getDarwinMemoryStatus() (MemoryStatus, error) {
	totalOut, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return MemoryStatus{}, err
	}
	total, err := strconv.ParseUint(strings.TrimSpace(string(totalOut)), 10, 64)
	if err != nil {
		return MemoryStatus{}, fmt.Errorf("parse hw.memsize: %w", err)
	}
	vmOut, err := exec.Command("vm_stat").Output()
	if err != nil {
		return MemoryStatus{}, err
	}
	return parseDarwinVmStat(string(vmOut), total)
}

func getDiskStatus() ([]DiskStatus, error) {
	if runtime.GOOS == "darwin" {
		output, err := exec.Command("df", "-kP").Output()
		if err != nil {
			return nil, err
		}
		return parsePOSIXDfKP(string(output), 1024), nil
	}
	output, err := exec.Command("df", "-B1", "--output=source,size,used,avail,target").Output()
	if err != nil {
		return nil, err
	}
	return parseGNUDfOutput(string(output)), nil
}

func getCPUStatus() (CPUStatus, error) {
	cpuStatus := CPUStatus{NumCPU: runtime.NumCPU()}
	if runtime.GOOS == "darwin" {
		output, err := exec.Command("top", "-l", "1", "-n", "0", "-s", "0").Output()
		if err != nil {
			return cpuStatus, err
		}
		cpuStatus.UsedPercent = parseDarwinTopCPU(string(output))
		return cpuStatus, nil
	}
	output, err := exec.Command("top", "-bn1").Output()
	if err != nil {
		return cpuStatus, err
	}
	cpuStatus.UsedPercent = parseLinuxTopCPU(string(output))
	return cpuStatus, nil
}

func getOSInfo() (OSInfo, error) {
	var osInfo OSInfo

	if runtime.GOOS == "darwin" {
		nameOut, _ := exec.Command("sw_vers", "-productName").Output()
		verOut, _ := exec.Command("sw_vers", "-productVersion").Output()
		name := strings.TrimSpace(string(nameOut))
		ver := strings.TrimSpace(string(verOut))
		switch {
		case name != "" && ver != "":
			osInfo.OS = name + " " + ver
		case name != "":
			osInfo.OS = name
		default:
			osInfo.OS = "Darwin"
		}
	} else if data, err := os.ReadFile("/etc/os-release"); err == nil {
		osInfo.OS = parsePrettyName(string(data))
	}

	if osInfo.OS == "" {
		cmd := exec.Command("uname", "-s")
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

	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.Command("ps", "aux")
	} else {
		cmd = exec.Command("ps", "aux", "--no-headers")
	}
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
		if fields[0] == "USER" || fields[0] == "UID" {
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
