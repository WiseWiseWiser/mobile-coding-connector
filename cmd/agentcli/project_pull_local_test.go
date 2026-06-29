package agentcli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNextWorktreeSuffix(t *testing.T) {
	parent := t.TempDir()
	got, err := nextWorktreeSuffix(parent, "main")
	if err != nil {
		t.Fatal(err)
	}
	if got != "main-1" {
		t.Fatalf("got %q", got)
	}
	if err := os.MkdirAll(filepath.Join(parent, "main-1"), 0755); err != nil {
		t.Fatal(err)
	}
	got2, err := nextWorktreeSuffix(parent, "main")
	if err != nil {
		t.Fatal(err)
	}
	if got2 != "main-2" {
		t.Fatalf("got %q", got2)
	}
}

func TestServerAndProjectSlug(t *testing.T) {
	if got := serverSlug("http://localhost:25001"); got != "localhost-25001" {
		t.Fatalf("server slug %q", got)
	}
	if got := projectSlug("pull-collision", "/tmp/foo"); got != "pull-collision" {
		t.Fatalf("project slug %q", got)
	}
}