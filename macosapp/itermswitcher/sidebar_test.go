package itermswitcher

import "testing"

func TestFormatSidebarTitle(t *testing.T) {
	cases := []struct {
		id, want string
	}{
		{SidebarAll, "All"},
		{"", "All"},
		{SidebarBookmarks, "Bookmarks"},
		{SidebarSaved, "Saved notes"},
		{SidebarDesktopID(2), "Desktop 3"},
		{"nope", ""},
	}
	for _, tc := range cases {
		if got := FormatSidebarTitle(tc.id); got != tc.want {
			t.Fatalf("FormatSidebarTitle(%q)=%q want %q", tc.id, got, tc.want)
		}
	}
}

func TestFilterSessionsBookmarks(t *testing.T) {
	sessions := fixtureSessions()
	out := FilterSessions(sessions, SidebarBookmarks, "")
	if len(out) != 2 {
		t.Fatalf("len=%d want 2", len(out))
	}
	if out[0].SessionID != "a" || out[1].SessionID != "c" {
		t.Fatalf("ids=%q %q", out[0].SessionID, out[1].SessionID)
	}
}

func TestFilterSessionsDesktopAndQuery(t *testing.T) {
	sessions := fixtureSessions()
	desk0 := FilterSessions(sessions, SidebarDesktopID(0), "")
	if len(desk0) != 2 {
		t.Fatalf("desktop 1 len=%d want 2", len(desk0))
	}
	auth := FilterSessions(sessions, SidebarAll, "auth")
	if len(auth) != 1 || auth[0].SessionID != "a" {
		t.Fatalf("query auth: %#v", auth)
	}
	saved := FilterSessions(sessions, SidebarSaved, "")
	if len(saved) != 0 {
		t.Fatalf("live rows must not match saved, got %d", len(saved))
	}
}

func TestCountBookmarked(t *testing.T) {
	if n := CountBookmarked(fixtureSessions()); n != 2 {
		t.Fatalf("count=%d want 2", n)
	}
}

func TestSidebarItems(t *testing.T) {
	items := SidebarItems([]int{0, 1}, 2, 1)
	if len(items) != 5 {
		t.Fatalf("len=%d want 5", len(items))
	}
	if items[0].ID != SidebarAll || items[1].ID != SidebarBookmarks {
		t.Fatalf("head=%q %q", items[0].ID, items[1].ID)
	}
	if items[2].Title != "Desktop 1" || items[3].Title != "Desktop 2" {
		t.Fatalf("desktops=%q %q", items[2].Title, items[3].Title)
	}
	if items[4].ID != SidebarSaved || items[4].Count != 1 {
		t.Fatalf("saved=%+v", items[4])
	}
}

func TestFormatOrphanPrimary(t *testing.T) {
	if got := FormatOrphanPrimary("cut last Friday", "old deploy", "~/d", "id"); got != "cut last Friday" {
		t.Fatalf("got %q", got)
	}
	if got := FormatOrphanPrimary("", "old deploy", "~/d", "id"); got != "old deploy" {
		t.Fatalf("got %q", got)
	}
}

func TestResolvedBookmarkedPrefersOverride(t *testing.T) {
	if !ResolvedBookmarked("a", false, map[string]bool{"a": true}) {
		t.Fatal("override true should win")
	}
	if ResolvedBookmarked("a", true, map[string]bool{"a": false}) {
		t.Fatal("override false should win")
	}
	if !ResolvedBookmarked("a", true, nil) {
		t.Fatal("nil overrides fall back to inventory")
	}
}

func TestReconcileBookmarkOverrides(t *testing.T) {
	live := map[string]bool{"a": true, "b": false}
	got := ReconcileBookmarkOverrides(map[string]bool{"a": true, "b": true, "gone": true}, live)
	if _, ok := got["a"]; ok {
		t.Fatal("caught-up override should drop")
	}
	if !got["b"] {
		t.Fatal("stale false inventory should keep override true")
	}
	if _, ok := got["gone"]; ok {
		t.Fatal("missing session should drop")
	}
}

func fixtureSessions() []FilterSession {
	return []FilterSession{
		{SessionID: "a", Name: "grok review", Note: "fix auth", SpaceIndex: 0, Bookmarked: true},
		{SessionID: "b", Name: "wrk build", SpaceIndex: 1, Bookmarked: false},
		{SessionID: "c", Name: "logs", Note: "tail prod", SpaceIndex: 0, Bookmarked: true},
	}
}
