package hostmon

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCPUAndDiskDelta(t *testing.T) {
	prev := counters{
		at:   time.Date(2026, 10, 21, 6, 40, 0, 0, time.FixedZone("CST", 8*3600)),
		cpus: 2,
		cpu:  cpuTicks{user: 100, system: 50, idle: 800, iowait: 50},
		disk: diskTicks{name: "vda", readIOs: 1000, readSectors: 2000, writeIOs: 10, writeSectors: 20},
		procs: map[int]procTicks{
			902: {pid: 902, comm: "node", readBytes: 0, writeBytes: 0, cpuTicks: 10},
		},
	}
	cur := counters{
		at:   prev.at.Add(15 * time.Second),
		cpus: 2,
		load: 6.2,
		cpu:  cpuTicks{user: 220, system: 80, idle: 950, iowait: 760},
		disk: diskTicks{name: "vda", readIOs: 1000 + 1804*15, readSectors: 2000 + (4404019/512)*15, writeIOs: 10 + 2*15, writeSectors: 20},
		procs: map[int]procTicks{
			902: {pid: 902, comm: "node", readBytes: uint64(1.1 * 1024 * 1024 * 15), writeBytes: 12 * 1024 * 15, cpuTicks: 10 + 200},
		},
	}
	sample := deltaSample(prev, cur, func(int) string { return "node server.js" })
	if sample.Disk.Dev != "vda" {
		t.Fatalf("disk = %q", sample.Disk.Dev)
	}
	if math.Abs(sample.Disk.RIops-1804) > 0.1 {
		t.Fatalf("riops = %v, want 1804", sample.Disk.RIops)
	}
	if math.Abs(sample.Disk.WIops-2) > 0.1 {
		t.Fatalf("wiops = %v, want 2", sample.Disk.WIops)
	}
	if sample.CPU.IOWait < 60 || sample.CPU.IOWait > 75 {
		t.Fatalf("iowait = %v, want about 71", sample.CPU.IOWait)
	}
	if len(sample.Top) != 1 || sample.Top[0].PID != 902 || sample.Top[0].Comm != "node" {
		t.Fatalf("top = %+v", sample.Top)
	}
	if sample.Top[0].Cmd != "node server.js" {
		t.Fatalf("cmd = %q", sample.Top[0].Cmd)
	}
	if sample.Top[0].ReadBps < 1.0*1024*1024 || sample.Top[0].ReadBps > 1.2*1024*1024 {
		t.Fatalf("readBps = %v", sample.Top[0].ReadBps)
	}
}

func TestSelectTopAddsHotCPU(t *testing.T) {
	procs := []ProcSample{
		{PID: 1, Comm: "a", ReadBps: 100},
		{PID: 2, Comm: "b", ReadBps: 90},
		{PID: 3, Comm: "c", ReadBps: 80},
		{PID: 4, Comm: "d", ReadBps: 70},
		{PID: 5, Comm: "e", ReadBps: 60},
		{PID: 6, Comm: "hot", ReadBps: 1, CPU: 40},
	}
	top := selectTop(procs)
	seen := map[int]bool{}
	for _, proc := range top {
		seen[proc.PID] = true
	}
	if !seen[6] {
		t.Fatalf("top = %+v, want pid 6 included for cpu", top)
	}
	if len(top) < 6 {
		t.Fatalf("top len = %d, want io top plus cpu", len(top))
	}
}

func TestReadProcTree(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "stat"), "cpu  100 0 50 800 50 0 0 0 0 0\ncpu0 50 0 25 400 25 0 0 0 0 0\ncpu1 50 0 25 400 25 0 0 0 0 0\n")
	mustWrite(t, filepath.Join(root, "loadavg"), "1.50 1.00 0.50 1/10 9\n")
	mustWrite(t, filepath.Join(root, "diskstats"), " 253 0 vda 10 0 20 0 1 0 2 0 0 0 0\n 7 0 loop0 99 0 99 0 0 0 0 0 0 0 0\n")
	pid := filepath.Join(root, "42")
	if err := os.Mkdir(pid, 0755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(pid, "io"), "read_bytes: 4096\nwrite_bytes: 0\n")
	mustWrite(t, filepath.Join(pid, "stat"), "42 (cloudflared) S 1 1 1 0 -1 0 0 0 0 0 5 1 0 0 20 0 1 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0\n")
	mustWrite(t, filepath.Join(pid, "comm"), "cloudflared\n")
	mustWrite(t, filepath.Join(pid, "cmdline"), "cloudflared\x00tunnel\x00run\x00")

	cur, err := readProc(root, time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	if cur.cpus != 2 || cur.load != 1.5 || cur.disk.name != "vda" {
		t.Fatalf("snapshot = %+v", cur)
	}
	if cur.procs[42].readBytes != 4096 || cur.procs[42].comm != "cloudflared" {
		t.Fatalf("proc = %+v", cur.procs[42])
	}
	if got := readCmdline(root, 42); got != "cloudflared tunnel run" {
		t.Fatalf("cmdline = %q", got)
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}
