package itermswitcher

import (
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/server/localiterm2"
)

func spaceLabelFixture() localiterm2.Inventory {
	return localiterm2.Inventory{
		ITermRunning: true,
		Desktops: []localiterm2.DesktopGroup{
			{
				SpaceIndex: 0,
				Desktop:    1,
				SpaceID:    10,
				Label:      "",
				Sessions: []localiterm2.LiveSession{
					{SessionID: "a", SessionName: "grok review", SpaceIndex: 0},
				},
			},
			{
				SpaceIndex: 1,
				Desktop:    2,
				SpaceID:    11,
				Label:      "Review staging",
				Sessions: []localiterm2.LiveSession{
					{SessionID: "b", SessionName: "wrk build", SpaceIndex: 1},
				},
			},
		},
	}
}

func TestViewAllHasNoSpaceLabelRow(t *testing.T) {
	s := NewUIState(spaceLabelFixture(), StatusCached)
	view := PaintView(s)
	if strings.Contains(view, "Set Space Label") {
		t.Fatalf("All must not show label row:\n%s", view)
	}
	if !strings.Contains(view, "grok review") {
		t.Fatalf("missing session:\n%s", view)
	}
}

func TestViewDesktopShowsLabelRowAndSeparator(t *testing.T) {
	s := NewUIState(spaceLabelFixture(), StatusCached)
	s, _ = ApplyKey(s, "]")
	view := PaintView(s)
	if !strings.Contains(view, "Set Space Label") {
		t.Fatalf("unset desktop missing prompt:\n%s", view)
	}
	if !strings.Contains(view, FormatSpaceLabelSeparator()) {
		t.Fatalf("missing separator:\n%s", view)
	}
	if !strings.Contains(view, "grok review") {
		t.Fatalf("missing session:\n%s", view)
	}
	if strings.Contains(view, "wrk build") {
		t.Fatalf("other space leaked:\n%s", view)
	}
}

func TestViewDesktopUsesLabelInSidebar(t *testing.T) {
	s := NewUIState(spaceLabelFixture(), StatusCached)
	s, _ = ApplyKey(s, "2")
	view := PaintView(s)
	if !strings.Contains(view, "Review staging") {
		t.Fatalf("missing labeled sidebar/row:\n%s", view)
	}
	if strings.Contains(view, "Set Space Label") {
		t.Fatalf("set desktop still shows prompt:\n%s", view)
	}
}

func TestEnterOnSpaceLabelRowIsNoop(t *testing.T) {
	s := NewUIState(spaceLabelFixture(), StatusCached)
	s, _ = ApplyKey(s, "]")
	s.ListIndex = 0
	_, act := ApplyKey(s, "enter")
	if act.Name != "" {
		t.Fatalf("enter on label row: %+v", act)
	}
}

func TestDownSkipsSeparatorToSession(t *testing.T) {
	s := NewUIState(spaceLabelFixture(), StatusCached)
	s, _ = ApplyKey(s, "]")
	s.ListIndex = 0
	s, _ = ApplyKey(s, "j")
	rows := listRows(s)
	if s.ListIndex >= len(rows) || rows[s.ListIndex].sessionID != "a" {
		t.Fatalf("listIndex=%d rows=%+v", s.ListIndex, rows)
	}
}
