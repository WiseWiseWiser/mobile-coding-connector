package server

import (
	"runtime"
	"strings"
	"testing"
)

func TestParseLinuxMeminfo(t *testing.T) {
	mem, err := parseLinuxMeminfo("MemTotal:       16384000 kB\nMemFree:         2048000 kB\nMemAvailable:    8192000 kB\n")
	if err != nil {
		t.Fatal(err)
	}
	if mem.Total != 16384000*1024 {
		t.Fatalf("total = %d", mem.Total)
	}
	if mem.Free != 2048000*1024 {
		t.Fatalf("free = %d", mem.Free)
	}
	wantUsed := uint64((16384000 - 8192000) * 1024)
	if mem.Used != wantUsed {
		t.Fatalf("used = %d, want %d", mem.Used, wantUsed)
	}
}

func TestParseDarwinVmStat(t *testing.T) {
	const total = 64 * 1024 * 1024 * 1024
	data := `Mach Virtual Memory Statistics: (page size of 16384 bytes)
Pages free:                               1000.
Pages active:                             2000.
Pages inactive:                           3000.
Pages speculative:                         400.
Pages wired down:                         5000.
Pages occupied by compressor:             6000.
`
	mem, err := parseDarwinVmStat(data, total)
	if err != nil {
		t.Fatal(err)
	}
	if mem.Total != total {
		t.Fatalf("total = %d", mem.Total)
	}
	page := uint64(16384)
	if mem.Free != 1000*page {
		t.Fatalf("free = %d", mem.Free)
	}
	if mem.Used == 0 || mem.Used >= total {
		t.Fatalf("used = %d", mem.Used)
	}
}

func TestParsePOSIXDfKP(t *testing.T) {
	out := `Filesystem 1024-blocks Used Available Capacity Mounted on
/dev/disk3s1s1  1000000  400000  600000   40%    /
devfs               200     200       0  100%    /dev
`
	disks := parsePOSIXDfKP(out, 1024)
	if len(disks) != 2 {
		t.Fatalf("len = %d", len(disks))
	}
	if disks[0].MountPoint != "/" || disks[0].Size != 1000000*1024 {
		t.Fatalf("%+v", disks[0])
	}
}

func TestParseGNUDfOutput(t *testing.T) {
	out := `Filesystem     Size Used Avail Mounted
/dev/sda1  1000 400 600 /
`
	disks := parseGNUDfOutput(out)
	if len(disks) != 1 || disks[0].MountPoint != "/" || disks[0].Size != 1000 {
		t.Fatalf("%+v", disks)
	}
}

func TestParseDarwinTopCPU(t *testing.T) {
	got := parseDarwinTopCPU("CPU usage: 8.62% user, 13.79% sys, 77.57% idle\n")
	if got < 22 || got > 23 {
		t.Fatalf("used = %v, want ~22.43", got)
	}
}

func TestParseLinuxTopCPU(t *testing.T) {
	got := parseLinuxTopCPU("%Cpu(s): 10.0 us,  5.0 sy,  0.0 ni, 85.0 id,  0.0 wa\n")
	if got < 14.9 || got > 15.1 {
		t.Fatalf("used = %v, want 15", got)
	}
}

func TestGetServerStatusDoesNotFailOnDarwinCollectors(t *testing.T) {
	st := getServerStatus()
	if st == nil {
		t.Fatal("nil status")
	}
	if runtime.GOOS == "darwin" && st.Memory.Total == 0 {
		t.Fatal("darwin memory total should be non-zero")
	}
	if st.OSInfo.Arch == "" {
		t.Fatal("arch")
	}
	if st.Disk == nil || st.TopCPU == nil || st.TopMem == nil {
		t.Fatal("nil slices")
	}
}

func TestParsePrettyName(t *testing.T) {
	got := parsePrettyName("NAME=ubuntu\nPRETTY_NAME=\"Ubuntu 24.04\"\n")
	if got != "Ubuntu 24.04" {
		t.Fatalf("got %q", got)
	}
}

func TestPsAuxSkipsHeader(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin ps aux")
	}
	procs, err := getTopProcesses(1, "cpu")
	if err != nil {
		t.Fatal(err)
	}
	if len(procs) == 0 {
		t.Fatal("expected at least one process")
	}
	if strings.EqualFold(procs[0].Name, "USER") {
		t.Fatalf("header leaked: %+v", procs[0])
	}
}
