package main

import (
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/client"
)

func TestMatchProjectTargetPrefersID(t *testing.T) {
	projects := []client.ProjectInfo{
		{ID: "p1", Name: "web", Dir: "/tmp/one"},
		{ID: "web", Name: "other", Dir: "/tmp/two"},
	}

	got, err := matchProjectTarget(projects, "web")
	if err != nil {
		t.Fatalf("matchProjectTarget() error = %v", err)
	}
	if got.ID != "web" {
		t.Fatalf("matchProjectTarget() ID = %q, want %q", got.ID, "web")
	}
}

func TestMatchProjectTargetRejectsAmbiguousName(t *testing.T) {
	projects := []client.ProjectInfo{
		{ID: "p1", Name: "web", Dir: "/tmp/one"},
		{ID: "p2", Name: "web", Dir: "/tmp/two"},
	}

	_, err := matchProjectTarget(projects, "web")
	if err == nil {
		t.Fatalf("matchProjectTarget() error = nil, want ambiguity error")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("matchProjectTarget() error = %q, want ambiguity message", err.Error())
	}
}

func TestMatchProjectTargetMatchesDir(t *testing.T) {
	projects := []client.ProjectInfo{
		{ID: "p1", Name: "web", Dir: "/tmp/one"},
	}

	got, err := matchProjectTarget(projects, "/tmp/one")
	if err != nil {
		t.Fatalf("matchProjectTarget() error = %v", err)
	}
	if got.ID != "p1" {
		t.Fatalf("matchProjectTarget() ID = %q, want %q", got.ID, "p1")
	}
}
