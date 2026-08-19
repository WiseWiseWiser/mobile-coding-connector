package localiterm2

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	kooliterm "github.com/xhd2015/kool/tools/iterm2"
)

func fourteenSpaces() []SpaceRef {
	out := make([]SpaceRef, 14)
	for i := range out {
		out[i] = SpaceRef{SpaceIndex: i, SpaceID: uint64(100 + i)}
	}
	return out
}

func TestInventoryFromSessionsDoesNotGrowPastDesktopCount(t *testing.T) {
	h := &Handler{
		CountDesktops: func() (int, error) { return 14, nil },
		ListSpaces: func() ([]SpaceRef, error) {
			out := make([]SpaceRef, 14)
			for i := range out {
				out[i] = SpaceRef{SpaceIndex: i, SpaceID: uint64(100 + i)}
			}
			return out, nil
		},
		SpaceLabels: NewSpaceLabelStore(t.TempDir() + "/space-labels.json"),
	}
	inv := h.inventoryFromSessions([]LiveSession{
		{SessionID: "live", SessionName: "ok", SpaceIndex: 3, Desktop: 4},
		{SessionID: "ghost", SessionName: "old", SpaceIndex: 15, Desktop: 16},
	}, true)
	if len(inv.Desktops) != 14 {
		t.Fatalf("desktops=%d want 14", len(inv.Desktops))
	}
	if inv.Desktops[len(inv.Desktops)-1].Desktop != 14 {
		t.Fatalf("last=%d", inv.Desktops[len(inv.Desktops)-1].Desktop)
	}
	ids := map[string]bool{}
	for _, d := range inv.Desktops {
		for _, s := range d.Sessions {
			ids[s.SessionID] = true
		}
	}
	if !ids["live"] {
		t.Fatal("missing remapped-in-range session")
	}
	if ids["ghost"] {
		t.Fatal("stale Desktop 16 session must not create a ghost Space")
	}
}

func TestBuildInventoryDoesNotEnsurePastListedDesktops(t *testing.T) {
	snap := &kooliterm.Snapshot{Windows: []kooliterm.SnapshotWindow{{
		WindowID: 42,
		Tabs: []kooliterm.SnapshotTab{{
			Index: 1,
			Sessions: []kooliterm.SnapshotSession{{
				ID:   "ghost",
				Name: "old",
			}},
		}},
	}}}
	inv := BuildInventory(snap, desktopsFromCount(14), map[uint64]int{42: 15}, emptyNotes(), true)
	if len(inv.Desktops) != 14 {
		t.Fatalf("desktops=%d want 14", len(inv.Desktops))
	}
	if inv.Desktops[13].Sessions[0].SessionID != "ghost" {
		t.Fatalf("out-of-range window should clamp to last listed desktop, got %+v", inv.Desktops[13].Sessions)
	}
}

func TestClipDesktopsDropsGhostAndAddsMissing(t *testing.T) {
	inv := Inventory{
		Desktops: []DesktopGroup{
			{SpaceIndex: 0, Desktop: 1, Sessions: []LiveSession{{SessionID: "a"}}},
			{SpaceIndex: 14, Desktop: 15, Sessions: []LiveSession{{SessionID: "ghost15"}}},
			{SpaceIndex: 15, Desktop: 16, Sessions: []LiveSession{{SessionID: "ghost16"}}},
		},
	}
	got := ClipDesktops(inv, []SpaceRef{
		{SpaceIndex: 0, SpaceID: 10},
		{SpaceIndex: 1, SpaceID: 11},
	})
	if len(got.Desktops) != 2 {
		t.Fatalf("desktops=%d want 2", len(got.Desktops))
	}
	if got.Desktops[0].SpaceIndex != 0 || got.Desktops[1].SpaceIndex != 1 {
		t.Fatalf("indexes=%d %d", got.Desktops[0].SpaceIndex, got.Desktops[1].SpaceIndex)
	}
	if got.Desktops[0].Sessions[0].SessionID != "a" {
		t.Fatal("kept in-range session")
	}
	if len(got.Desktops[1].Sessions) != 0 {
		t.Fatalf("new live space should be empty, got %+v", got.Desktops[1].Sessions)
	}
}

func TestClipDesktopsEmptyLiveLeavesInventory(t *testing.T) {
	inv := Inventory{Desktops: []DesktopGroup{{SpaceIndex: 15, Desktop: 16}}}
	got := ClipDesktops(inv, nil)
	if len(got.Desktops) != 1 || got.Desktops[0].Desktop != 16 {
		t.Fatalf("empty live must not clip: %+v", got.Desktops)
	}
}

func TestMergeLayoutRestampsSpaceIndex(t *testing.T) {
	dir := t.TempDir()
	h := &Handler{
		CountDesktops:  func() (int, error) { return 14, nil },
		SpaceForWindow: func(windowID uint64) (int, error) { return 12, nil },
		ListSpaces:     func() ([]SpaceRef, error) { return fourteenSpaces(), nil },
		Notes:          NewNoteStore(filepath.Join(dir, "notes.json")),
		SpaceLabels:    NewSpaceLabelStore(filepath.Join(dir, "space-labels.json")),
	}
	last := Inventory{
		ITermRunning: true,
		Desktops: []DesktopGroup{{
			SpaceIndex: 15,
			Desktop:    16,
			Sessions:   []LiveSession{{SessionID: "a", WindowID: "42", SpaceIndex: 15, Desktop: 16, SessionName: "old"}},
		}},
	}
	layout := &kooliterm.Snapshot{Windows: []kooliterm.SnapshotWindow{{
		WindowID: 42,
		Tabs: []kooliterm.SnapshotTab{{
			Index:    1,
			Sessions: []kooliterm.SnapshotSession{{ID: "a", Name: "old"}},
		}},
	}}}
	got := h.mergeLayout(last, layout, nil, true)
	if len(got.Desktops) != 14 {
		t.Fatalf("desktops=%d want 14", len(got.Desktops))
	}
	found := false
	for _, d := range got.Desktops {
		for _, s := range d.Sessions {
			if s.SessionID != "a" {
				continue
			}
			found = true
			if d.SpaceIndex != 12 || s.SpaceIndex != 12 || s.Desktop != 13 {
				t.Fatalf("session a on desktop %+v session %+v", d.SpaceIndex, s)
			}
		}
	}
	if !found {
		t.Fatal("session a missing after rematch")
	}
}

func TestCachedInventoryRewritesClippedHeadings(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "iterm-inventory-cache.json")
	stale := Inventory{
		ITermRunning: true,
		Desktops: []DesktopGroup{
			{SpaceIndex: 0, Desktop: 1, Sessions: []LiveSession{{SessionID: "a", SessionName: "keep"}}},
			{SpaceIndex: 15, Desktop: 16, Sessions: []LiveSession{{SessionID: "ghost", SessionName: "old"}}},
		},
	}
	data, err := json.Marshal(stale)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		t.Fatal(err)
	}
	h := &Handler{
		CachePath:     cachePath,
		CountDesktops: func() (int, error) { return 14, nil },
		ListSpaces:    func() ([]SpaceRef, error) { return fourteenSpaces(), nil },
		Notes:         NewNoteStore(filepath.Join(dir, "notes.json")),
		SpaceLabels:   NewSpaceLabelStore(filepath.Join(dir, "space-labels.json")),
	}
	got, ok := h.CachedInventory()
	if !ok {
		t.Fatal("expected cache hit")
	}
	if len(got.Desktops) != 14 {
		t.Fatalf("response desktops=%d want 14", len(got.Desktops))
	}
	raw, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	var disk Inventory
	if err := json.Unmarshal(raw, &disk); err != nil {
		t.Fatal(err)
	}
	if len(disk.Desktops) != 14 {
		t.Fatalf("cache file desktops=%d want 14", len(disk.Desktops))
	}
}

func markSnap(id string, pid int, tabName string) *kooliterm.Snapshot {
	p := pid
	return &kooliterm.Snapshot{Windows: []kooliterm.SnapshotWindow{{
		WindowID: 1,
		Name:     "win",
		Tabs: []kooliterm.SnapshotTab{{
			Index: 1,
			Name:  tabName,
			Sessions: []kooliterm.SnapshotSession{{
				ID:   id,
				Name: tabName,
				PID:  &p,
			}},
		}},
	}}}
}

func TestBuildInventoryJoinsMark(t *testing.T) {
	resolve := func(pid int) (string, error) {
		if pid == 42 {
			return "waiting for CI", nil
		}
		return "", os.ErrNotExist
	}
	inv := buildInventory(markSnap("s1", 42, "mark"), desktopsFromCount(1), map[uint64]int{1: 0}, emptyNotes(), true, resolve)
	if len(inv.Desktops) == 0 || len(inv.Desktops[0].Sessions) == 0 {
		t.Fatalf("no session: %+v", inv)
	}
	got := inv.Desktops[0].Sessions[0]
	if got.PID != 42 {
		t.Fatalf("pid=%d", got.PID)
	}
	if got.Mark != "waiting for CI" {
		t.Fatalf("mark=%q", got.Mark)
	}
}

func TestBuildInventoryMarkMissStaysEmpty(t *testing.T) {
	resolve := func(int) (string, error) { return "", os.ErrNotExist }
	inv := buildInventory(markSnap("s1", 99, "mark"), desktopsFromCount(1), map[uint64]int{1: 0}, emptyNotes(), true, resolve)
	got := inv.Desktops[0].Sessions[0]
	if got.PID != 99 {
		t.Fatalf("pid=%d", got.PID)
	}
	if got.Mark != "" {
		t.Fatalf("mark=%q", got.Mark)
	}
}

func TestBuildInventoryDoesNotResolveMarkWithoutHook(t *testing.T) {
	inv := BuildInventory(markSnap("s1", 42, "mark"), desktopsFromCount(1), map[uint64]int{1: 0}, emptyNotes(), true)
	got := inv.Desktops[0].Sessions[0]
	if got.PID != 42 {
		t.Fatalf("pid=%d", got.PID)
	}
	if got.Mark != "" {
		t.Fatalf("public BuildInventory must not read ~/.mark: %q", got.Mark)
	}
}

func TestAssembleInventoryUsesHandlerResolveMark(t *testing.T) {
	h := &Handler{
		ResolveMark: func(pid int) (string, error) {
			if pid == 7 {
				return "still here", nil
			}
			return "", os.ErrNotExist
		},
	}
	inv := h.assembleInventory(markSnap("s1", 7, "mark"), desktopsFromCount(1), map[uint64]int{1: 0}, emptyNotes(), true)
	if inv.Desktops[0].Sessions[0].Mark != "still here" {
		t.Fatalf("mark=%q", inv.Desktops[0].Sessions[0].Mark)
	}
}

func TestMergeSpaceSessionsReplacesOnlyThatDesktop(t *testing.T) {
	last := Inventory{
		ITermRunning: true,
		Desktops: []DesktopGroup{
			{SpaceIndex: 0, Desktop: 1, Sessions: []LiveSession{{SessionID: "keep", Mark: "other"}}},
			{SpaceIndex: 10, Desktop: 11, Sessions: []LiveSession{{SessionID: "stale", PID: 31954, Mark: "allow the double click label to edit it, and allow drag to move it"}}},
		},
	}
	cap := Inventory{
		Desktops: []DesktopGroup{
			{SpaceIndex: 10, Desktop: 11, Sessions: []LiveSession{{SessionID: "stale", SessionName: "Default (-bash)", TabName: "Default (-bash)"}}},
		},
	}
	got := mergeSpaceSessions(last, 10, cap)
	if got.Desktops[0].Sessions[0].SessionID != "keep" || got.Desktops[0].Sessions[0].Mark != "other" {
		t.Fatalf("other space mutated: %+v", got.Desktops[0].Sessions)
	}
	s := got.Desktops[1].Sessions[0]
	if s.Mark != "" || s.TabName != "Default (-bash)" || s.SessionID != "stale" {
		t.Fatalf("space 10 not replaced: %+v", s)
	}
}

func TestRefreshSpaceRecapturesMarks(t *testing.T) {
	dir := t.TempDir()
	h := &Handler{
		CountDesktops: func() (int, error) { return 2, nil },
		ListSpaces: func() ([]SpaceRef, error) {
			return []SpaceRef{
				{SpaceIndex: 0, SpaceID: 100},
				{SpaceIndex: 1, SpaceID: 101},
			}, nil
		},
		SpaceForWindow: func(windowID uint64) (int, error) { return 1, nil },
		Notes:          NewNoteStore(filepath.Join(dir, "notes.json")),
		SpaceLabels:    NewSpaceLabelStore(filepath.Join(dir, "space-labels.json")),
		ResolveMark: func(pid int) (string, error) {
			if pid == 75470 {
				return "persist bookmark", nil
			}
			return "", os.ErrNotExist
		},
		CaptureSpace: func(spaceIndex int) (*kooliterm.Snapshot, error) {
			if spaceIndex != 1 {
				t.Fatalf("spaceIndex=%d", spaceIndex)
			}
			p := 75470
			return &kooliterm.Snapshot{Windows: []kooliterm.SnapshotWindow{{
				WindowID: 9,
				Name:     "win",
				Tabs: []kooliterm.SnapshotTab{{
					Index: 1,
					Name:  "Default (mark)",
					Sessions: []kooliterm.SnapshotSession{{
						ID:   "live",
						Name: "Default (mark)",
						PID:  &p,
					}},
				}},
			}}}, nil
		},
	}
	h.storeCache(Inventory{
		ITermRunning: true,
		Desktops: []DesktopGroup{
			{SpaceIndex: 0, Desktop: 1, SpaceID: 100, Sessions: []LiveSession{{SessionID: "other", Mark: "keep-me"}}},
			{SpaceIndex: 1, Desktop: 2, SpaceID: 101, Sessions: []LiveSession{{
				SessionID: "gone",
				PID:       31954,
				TabName:   "Default (mark)",
				Mark:      "allow the double click label to edit it, and allow drag to move it",
			}}},
		},
	})
	got, err := h.RefreshSpaceID(101)
	if err != nil {
		t.Fatal(err)
	}
	var other, live *LiveSession
	for _, d := range got.Desktops {
		for i := range d.Sessions {
			s := &d.Sessions[i]
			switch s.SessionID {
			case "other":
				other = s
			case "live":
				live = s
			case "gone":
				t.Fatal("stale session still present")
			}
		}
	}
	if other == nil || other.Mark != "keep-me" {
		t.Fatalf("other space: %+v", other)
	}
	if live == nil || live.Mark != "persist bookmark" || live.PID != 75470 {
		t.Fatalf("refreshed space: %+v", live)
	}
}
