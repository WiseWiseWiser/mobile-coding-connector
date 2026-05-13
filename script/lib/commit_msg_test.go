package lib

import "testing"

func TestShellQuoteNewlinesUseSingleAnsiEscape(t *testing.T) {
	got := shellQuote("Add metrics\n\nTrack averages")
	want := "$'Add metrics\\n\\nTrack averages'"
	if got != want {
		t.Fatalf("shellQuote() = %q, want %q", got, want)
	}
}

func TestShellQuoteEscapesLiteralBackslashN(t *testing.T) {
	got := shellQuote(`Keep literal \n sequence`)
	want := `$'Keep literal \\n sequence'`
	if got != want {
		t.Fatalf("shellQuote() = %q, want %q", got, want)
	}
}
