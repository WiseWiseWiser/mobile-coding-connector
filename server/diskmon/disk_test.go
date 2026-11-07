package diskmon

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func entry(mount, device, fsType string, blocks, bfree, bavail, bsize uint64, ro bool) mountEntry {
	return mountEntry{
		Mount:    mount,
		Device:   device,
		FSType:   fsType,
		Blocks:   blocks,
		Bfree:    bfree,
		Bavail:   bavail,
		Bsize:    bsize,
		ReadOnly: ro,
	}
}

func fixedNow() time.Time {
	return time.Date(2026, 11, 7, 10, 30, 0, 0, time.UTC)
}

func TestDarwinRootMergesIntoData(t *testing.T) {
	entries := []mountEntry{
		entry("/", "/dev/disk3s1s1", "apfs", 500000, 100000, 90000, 4096, true),
		entry("/System/Volumes/Data", "/dev/disk3s5", "apfs", 500000, 100000, 90000, 4096, false),
	}
	snap := buildSnapshot(entries, fixedNow())

	if len(snap.Volumes) != 1 {
		t.Fatalf("want 1 counted volume, got %d: %+v", len(snap.Volumes), snap.Volumes)
	}
	if len(snap.Merged) != 1 || snap.Merged[0].Mount != "/" {
		t.Fatalf("want / merged into Data, got %+v", snap.Merged)
	}
	if snap.PrimaryMount != dataMount {
		t.Fatalf("primary mount = %q, want %q", snap.PrimaryMount, dataMount)
	}
	if !snap.Volumes[0].Primary {
		t.Fatal("data volume must be primary")
	}
	total := uint64(500000) * 4096
	if snap.Volumes[0].TotalBytes != int64(total) {
		t.Fatalf("total = %d, want %d", snap.Volumes[0].TotalBytes, total)
	}
	if snap.Volumes[0].FreeBytes != int64(90000*4096) {
		t.Fatalf("free = %d, want avail*bsize", snap.Volumes[0].FreeBytes)
	}
	if snap.Volumes[0].UsedBytes != int64((500000-100000)*4096) {
		t.Fatalf("used = %d, want (blocks-bfree)*bsize", snap.Volumes[0].UsedBytes)
	}
	if snap.Volumes[0].UsedPercent != 80 {
		t.Fatalf("percent = %v, want 80", snap.Volumes[0].UsedPercent)
	}
}

func TestPseudoAndSupportMountsFiltered(t *testing.T) {
	entries := []mountEntry{
		entry("/", "/dev/disk1s1", "apfs", 100, 40, 30, 4096, false),
		entry("/dev", "devfs", "devfs", 0, 0, 0, 0, false),
		entry("/System/Volumes/VM", "/dev/disk1s4", "apfs", 100, 90, 80, 4096, false),
		entry("/private/var/vm", "/dev/disk1s4", "apfs", 100, 90, 80, 4096, false),
		entry("/proc", "proc", "proc", 0, 0, 0, 0, false),
		entry("/snap/core/1", "/dev/loop3", "squashfs", 0, 0, 0, 0, true),
		entry("/", "map auto_home", "autofs", 0, 0, 0, 0, false),
		entry("/Library/Developer/CoreSimulator/Volumes/iOS_22F77", "/dev/disk7s1", "apfs", 100, 90, 1, 4096, true),
	}
	snap := buildSnapshot(entries, fixedNow())
	if len(snap.Volumes) != 1 || snap.Volumes[0].Mount != "/" {
		t.Fatalf("want only /, got %+v", snap.Volumes)
	}
}

func TestNetworkMountsSurvive(t *testing.T) {
	entries := []mountEntry{
		entry("/", "/dev/disk1s1", "apfs", 100, 40, 30, 4096, false),
		entry("/Volumes/share", "nas:/export", "nfs", 200, 100, 90, 4096, false),
		entry("/Volumes/smb", "//user@host/share", "smbfs", 300, 200, 150, 4096, false),
	}
	snap := buildSnapshot(entries, fixedNow())
	if len(snap.Volumes) != 3 {
		t.Fatalf("network mounts must survive, got %+v", snap.Volumes)
	}
	if snap.Volumes[0].Mount != "/" || !snap.Volumes[0].Primary {
		t.Fatalf("primary volume must sort first, got %+v", snap.Volumes[0])
	}
}

func TestZeroCapacityDropsVolumeAndWarns(t *testing.T) {
	entries := []mountEntry{
		entry("/", "/dev/disk1s1", "apfs", 100, 40, 30, 4096, false),
		entry("/mnt/broken", "/dev/disk9", "ext4", 0, 0, 0, 0, false),
	}
	snap := buildSnapshot(entries, fixedNow())
	if len(snap.Volumes) != 1 {
		t.Fatalf("zero-capacity volume must be dropped, got %+v", snap.Volumes)
	}
	if snap.Warning == "" {
		t.Fatal("unreadable mount must produce a warning")
	}
	for _, v := range snap.Volumes {
		if v.UsedPercent < 0 || v.UsedPercent > 100 {
			t.Fatalf("percent out of range: %+v", v)
		}
	}
}

func TestNoVolumesNoPrimary(t *testing.T) {
	snap := buildSnapshot(nil, fixedNow())
	if len(snap.Volumes) != 0 || snap.PrimaryMount != "" {
		t.Fatalf("empty mount table must stay empty, got %+v", snap)
	}
	if snap.Volumes == nil {
		t.Fatal("volumes must marshal as [] not null")
	}
}

func TestPercentOf(t *testing.T) {
	cases := []struct {
		used, total uint64
		want        float64
	}{
		{0, 100, 0},
		{799, 1000, 79.9},
		{8, 10, 80},
		{949, 1000, 94.9},
		{95, 100, 95},
		{100, 100, 100},
		{1, 0, 0},
	}
	for _, c := range cases {
		if got := percentOf(c.used, c.total); got != c.want {
			t.Fatalf("percentOf(%d, %d) = %v, want %v", c.used, c.total, got, c.want)
		}
	}
}

func TestSnapshotJSONOmitsUnavailableIO(t *testing.T) {
	snap := buildSnapshot([]mountEntry{
		entry("/", "/dev/disk1s1", "apfs", 100, 40, 30, 4096, false),
	}, fixedNow())
	raw, err := json.Marshal(snap)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if _, ok := decoded["io"]; ok {
		t.Fatalf("io must be absent when hostmon has no sample: %s", raw)
	}
	if decoded["primary_mount"] != "/" {
		t.Fatalf("primary_mount = %v", decoded["primary_mount"])
	}
}

func TestSamplerCachesAndAttachesIO(t *testing.T) {
	calls := 0
	clock := fixedNow()
	s := &sampler{env: Env{
		Now: func() time.Time { return clock },
		ReadMounts: func() ([]mountEntry, error) {
			calls++
			return []mountEntry{entry("/", "/dev/disk1s1", "apfs", 100, 40, 30, 4096, false)}, nil
		},
		LatestIO: func() (IO, bool) {
			return IO{Device: "sda", ReadBps: 1024, WriteBps: 2048, IntervalSec: 15}, true
		},
	}}

	first, err := s.get()
	if err != nil {
		t.Fatal(err)
	}
	if first.IO == nil || first.IO.ReadBps != 1024 {
		t.Fatalf("io not attached: %+v", first.IO)
	}
	if _, err := s.get(); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("second get within TTL must hit cache, mount reads = %d", calls)
	}

	clock = clock.Add(cacheTTL)
	if _, err := s.get(); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("get after TTL must re-read, mount reads = %d", calls)
	}
}

func TestHandleDiskRejectsNonGet(t *testing.T) {
	mux := http.NewServeMux()
	RegisterAPI(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/server/disk", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestHandleDiskServesSnapshot(t *testing.T) {
	prev := defaultSampler
	defaultSampler = &sampler{env: Env{
		Now: fixedNow,
		ReadMounts: func() ([]mountEntry, error) {
			return []mountEntry{entry("/", "/dev/disk1s1", "apfs", 100, 40, 30, 4096, false)}, nil
		},
		LatestIO: func() (IO, bool) { return IO{}, false },
	}}
	defer func() { defaultSampler = prev }()

	mux := http.NewServeMux()
	RegisterAPI(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/server/disk", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var snap Snapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &snap); err != nil {
		t.Fatal(err)
	}
	if snap.PrimaryMount != "/" || len(snap.Volumes) != 1 {
		t.Fatalf("unexpected snapshot: %+v", snap)
	}
	if snap.SampledAt != fixedNow().Format(time.RFC3339) {
		t.Fatalf("sampled_at = %q", snap.SampledAt)
	}
}

func TestUnescapeMount(t *testing.T) {
	cases := map[string]string{
		`/Volumes/My\040Disk`: "/Volumes/My Disk",
		"/plain":              "/plain",
		`/a\134b`:             `/a\b`,
		`/t\011x`:             "/t\tx",
	}
	for in, want := range cases {
		if got := unescapeMount(in); got != want {
			t.Fatalf("unescapeMount(%q) = %q, want %q", in, got, want)
		}
	}
}
