package hostmon

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type cpuTicks struct {
	user, nice, system, idle, iowait, irq, softirq, steal uint64
}

func (c cpuTicks) total() uint64 {
	return c.user + c.nice + c.system + c.idle + c.iowait + c.irq + c.softirq + c.steal
}

type diskTicks struct {
	name         string
	readIOs      uint64
	readSectors  uint64
	writeIOs     uint64
	writeSectors uint64
}

type procTicks struct {
	pid        int
	comm       string
	readBytes  uint64
	writeBytes uint64
	cpuTicks   uint64
}

type counters struct {
	at    time.Time
	cpus  int
	load  float64
	cpu   cpuTicks
	disk  diskTicks
	procs map[int]procTicks
}

// readProc collects one absolute snapshot from a /proc-shaped directory.
func readProc(root string, now time.Time) (counters, error) {
	statText, err := os.ReadFile(filepath.Join(root, "stat"))
	if err != nil {
		return counters{}, err
	}
	cpu, ncpu, err := parseCPUStat(string(statText))
	if err != nil {
		return counters{}, err
	}
	loadText, err := os.ReadFile(filepath.Join(root, "loadavg"))
	if err != nil {
		return counters{}, err
	}
	load, err := parseLoad(string(loadText))
	if err != nil {
		return counters{}, err
	}
	diskText, err := os.ReadFile(filepath.Join(root, "diskstats"))
	if err != nil {
		return counters{}, err
	}
	disk := pickDisk(parseDiskstats(string(diskText)))
	procs := readProcs(root)
	return counters{at: now, cpus: ncpu, load: load, cpu: cpu, disk: disk, procs: procs}, nil
}

func parseCPUStat(text string) (cpuTicks, int, error) {
	var agg cpuTicks
	ncpu := 0
	found := false
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 || !strings.HasPrefix(fields[0], "cpu") {
			continue
		}
		ticks, err := parseCPUFields(fields[1:])
		if err != nil {
			return cpuTicks{}, 0, err
		}
		if fields[0] == "cpu" {
			agg = ticks
			found = true
			continue
		}
		ncpu++
	}
	if !found {
		return cpuTicks{}, 0, fmt.Errorf("proc stat: no cpu line")
	}
	if ncpu == 0 {
		ncpu = 1
	}
	return agg, ncpu, nil
}

func parseCPUFields(fields []string) (cpuTicks, error) {
	nums := make([]uint64, 8)
	for i := 0; i < len(nums) && i < len(fields); i++ {
		n, err := strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			return cpuTicks{}, fmt.Errorf("proc stat: %w", err)
		}
		nums[i] = n
	}
	return cpuTicks{
		user: nums[0], nice: nums[1], system: nums[2], idle: nums[3],
		iowait: nums[4], irq: nums[5], softirq: nums[6], steal: nums[7],
	}, nil
}

func parseLoad(text string) (float64, error) {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return 0, fmt.Errorf("proc loadavg: empty")
	}
	load, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, fmt.Errorf("proc loadavg: %w", err)
	}
	return load, nil
}

func parseDiskstats(text string) []diskTicks {
	var out []diskTicks
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 11 {
			continue
		}
		name := fields[2]
		if skipDisk(name) {
			continue
		}
		readIOs, _ := strconv.ParseUint(fields[3], 10, 64)
		readSectors, _ := strconv.ParseUint(fields[5], 10, 64)
		writeIOs, _ := strconv.ParseUint(fields[7], 10, 64)
		writeSectors, _ := strconv.ParseUint(fields[9], 10, 64)
		out = append(out, diskTicks{
			name: name, readIOs: readIOs, readSectors: readSectors,
			writeIOs: writeIOs, writeSectors: writeSectors,
		})
	}
	return out
}

func skipDisk(name string) bool {
	switch {
	case strings.HasPrefix(name, "loop"), strings.HasPrefix(name, "ram"):
		return true
	case strings.HasPrefix(name, "dm-"), strings.HasPrefix(name, "md"):
		return true
	case strings.HasPrefix(name, "fd"):
		return true
	default:
		return false
	}
}

func pickDisk(disks []diskTicks) diskTicks {
	for _, want := range []string{"vda", "sda", "nvme0n1"} {
		for _, disk := range disks {
			if disk.name == want {
				return disk
			}
		}
	}
	if len(disks) == 0 {
		return diskTicks{}
	}
	return disks[0]
}

func readProcs(root string) map[int]procTicks {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	out := make(map[int]procTicks)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 {
			continue
		}
		base := filepath.Join(root, entry.Name())
		ioText, err := os.ReadFile(filepath.Join(base, "io"))
		if err != nil {
			continue
		}
		statText, err := os.ReadFile(filepath.Join(base, "stat"))
		if err != nil {
			continue
		}
		readB, writeB := parseProcIO(string(ioText))
		ticks, ok := parseProcStatTicks(string(statText))
		if !ok {
			continue
		}
		commBytes, _ := os.ReadFile(filepath.Join(base, "comm"))
		out[pid] = procTicks{
			pid: pid, comm: strings.TrimSpace(string(commBytes)),
			readBytes: readB, writeBytes: writeB, cpuTicks: ticks,
		}
	}
	return out
}

func parseProcIO(text string) (readBytes, writeBytes uint64) {
	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		key, val, ok := strings.Cut(sc.Text(), ":")
		if !ok {
			continue
		}
		n, err := strconv.ParseUint(strings.TrimSpace(val), 10, 64)
		if err != nil {
			continue
		}
		switch strings.TrimSpace(key) {
		case "read_bytes":
			readBytes = n
		case "write_bytes":
			writeBytes = n
		}
	}
	return readBytes, writeBytes
}

func parseProcStatTicks(text string) (uint64, bool) {
	end := strings.LastIndex(text, ")")
	if end < 0 || end+2 >= len(text) {
		return 0, false
	}
	fields := strings.Fields(text[end+2:])
	if len(fields) < 13 {
		return 0, false
	}
	utime, err1 := strconv.ParseUint(fields[11], 10, 64)
	stime, err2 := strconv.ParseUint(fields[12], 10, 64)
	if err1 != nil || err2 != nil {
		return 0, false
	}
	return utime + stime, true
}

func readCmdline(root string, pid int) string {
	raw, err := os.ReadFile(filepath.Join(root, strconv.Itoa(pid), "cmdline"))
	if err != nil || len(raw) == 0 {
		return ""
	}
	if len(raw) > 256 {
		raw = raw[:256]
	}
	raw = bytes.ReplaceAll(raw, []byte{0}, []byte{' '})
	return trimCmd(strings.TrimSpace(string(raw)))
}

func trimCmd(cmd string) string {
	cmd = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, cmd)
	if len(cmd) <= CmdMax {
		return cmd
	}
	return cmd[:CmdMax]
}

func deltaSample(prev, cur counters, cmdline func(pid int) string) Sample {
	dt := cur.at.Sub(prev.at).Seconds()
	if dt <= 0 {
		dt = 1
	}
	cpuDelta := float64(ticksDelta(cur.cpu.total(), prev.cpu.total()))
	sample := Sample{
		T:     cur.at.Format(time.RFC3339),
		Load1: cur.load,
		Disk: DiskSample{
			Dev:   cur.disk.name,
			RIops: rate(cur.disk.readIOs, prev.disk.readIOs, dt),
			WIops: rate(cur.disk.writeIOs, prev.disk.writeIOs, dt),
			RBps:  rate(cur.disk.readSectors, prev.disk.readSectors, dt) * 512,
			WBps:  rate(cur.disk.writeSectors, prev.disk.writeSectors, dt) * 512,
		},
	}
	if cpuDelta > 0 {
		sample.CPU = CPUSample{
			User:   100 * float64(ticksDelta(cur.cpu.user, prev.cpu.user)) / cpuDelta,
			System: 100 * float64(ticksDelta(cur.cpu.system, prev.cpu.system)) / cpuDelta,
			IOWait: 100 * float64(ticksDelta(cur.cpu.iowait, prev.cpu.iowait)) / cpuDelta,
			Idle:   100 * float64(ticksDelta(cur.cpu.idle, prev.cpu.idle)) / cpuDelta,
		}
	}
	var procs []ProcSample
	for pid, now := range cur.procs {
		was, ok := prev.procs[pid]
		if !ok {
			continue
		}
		procCPU := 0.0
		if cpuDelta > 0 {
			procCPU = 100 * float64(ticksDelta(now.cpuTicks, was.cpuTicks)) / cpuDelta * float64(cur.cpus)
		}
		procs = append(procs, ProcSample{
			PID:      pid,
			Comm:     now.comm,
			ReadBps:  rate(now.readBytes, was.readBytes, dt),
			WriteBps: rate(now.writeBytes, was.writeBytes, dt),
			CPU:      procCPU,
		})
	}
	sample.Top = selectTop(procs)
	for i := range sample.Top {
		if cmdline != nil {
			sample.Top[i].Cmd = cmdline(sample.Top[i].PID)
		}
	}
	return sample
}

func ticksDelta(now, was uint64) uint64 {
	if now < was {
		return 0
	}
	return now - was
}

func rate(now, was uint64, dt float64) float64 {
	if now < was || dt <= 0 {
		return 0
	}
	return float64(now-was) / dt
}

func selectTop(procs []ProcSample) []ProcSample {
	sort.Slice(procs, func(i, j int) bool {
		return procs[i].ReadBps+procs[i].WriteBps > procs[j].ReadBps+procs[j].WriteBps
	})
	out := make([]ProcSample, 0, TopN+2)
	seen := map[int]bool{}
	for _, proc := range procs {
		if len(out) >= TopN {
			break
		}
		if proc.ReadBps+proc.WriteBps <= 0 && proc.CPU < CPUExtraPercent {
			continue
		}
		out = append(out, proc)
		seen[proc.PID] = true
	}
	sort.Slice(procs, func(i, j int) bool { return procs[i].CPU > procs[j].CPU })
	for _, proc := range procs {
		if proc.CPU < CPUExtraPercent || seen[proc.PID] {
			continue
		}
		out = append(out, proc)
		seen[proc.PID] = true
	}
	return out
}
