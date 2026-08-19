package itermswitcher

import "testing"

func TestFormatSpaceLabelRow(t *testing.T) {
	if got := FormatSpaceLabelRow(""); got != "Set Space Label" {
		t.Fatalf("empty=%q", got)
	}
	if got := FormatSpaceLabelRow("  "); got != "Set Space Label" {
		t.Fatalf("ws=%q", got)
	}
	if got := FormatSpaceLabelRow(" Review staging "); got != "Review staging" {
		t.Fatalf("set=%q", got)
	}
}

func TestFormatSidebarDesktopTitle(t *testing.T) {
	if got := FormatSidebarDesktopTitle(2, ""); got != "Desktop 3" {
		t.Fatalf("fallback=%q", got)
	}
	if got := FormatSidebarDesktopTitle(2, "Review staging"); got != "Review staging" {
		t.Fatalf("label=%q", got)
	}
}

func TestFormatChangeAndClear(t *testing.T) {
	if FormatChangeSpaceLabel() != "Change" {
		t.Fatal(FormatChangeSpaceLabel())
	}
	if FormatClearSpaceLabel() != "Clear" {
		t.Fatal(FormatClearSpaceLabel())
	}
}
