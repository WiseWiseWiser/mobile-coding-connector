package textconvert

import (
	"strings"
	"testing"
)

func TestShellSingleLineExample(t *testing.T) {
	in := "setup-shpdev-cloudflared-mapping-via-xdev mapping create admin-api \\\n" +
		"  --url https://admin.example.com/api/ \\\n" +
		"  --port 18080 \\\n" +
		"  --yes"
	want := "setup-shpdev-cloudflared-mapping-via-xdev mapping create admin-api --url https://admin.example.com/api/ --port 18080 --yes"
	if got := ShellSingleLine(in); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestShellSingleLineCRLF(t *testing.T) {
	in := "cmd \\\r\n  --flag"
	want := "cmd --flag"
	if got := ShellSingleLine(in); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestShellSingleLineWhitespaceOnly(t *testing.T) {
	if got := ShellSingleLine("  \n\t  "); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestShellSingleLineTabsBlankLines(t *testing.T) {
	in := "a\t\tb\n\n  c   d"
	want := "a b c d"
	if got := ShellSingleLine(in); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestShellSingleLinePreservesNonContinuationBackslash(t *testing.T) {
	in := `path\to\file and C:\Windows`
	want := `path\to\file and C:\Windows`
	if got := ShellSingleLine(in); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestConvertDefaultAndUnknown(t *testing.T) {
	got, err := Convert("a \\\n b", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "a b" {
		t.Fatalf("got %q", got)
	}
	_, err = Convert("x", "no-such")
	if err == nil || !strings.Contains(err.Error(), "unknown converter") {
		t.Fatalf("err %v", err)
	}
}

func TestKnownConvertersIncludesShellSingleLine(t *testing.T) {
	ids := KnownConverters()
	found := false
	for _, id := range ids {
		if id == ConverterShellSingleLine {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing %s in %v", ConverterShellSingleLine, ids)
	}
}
