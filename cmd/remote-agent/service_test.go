package main

import (
	"strings"
	"testing"

	"github.com/xhd2015/lifelog-private/ai-critic/client"
)

func TestMatchServiceTargetPrefersID(t *testing.T) {
	services := []client.ServiceStatus{
		{ID: "svc-1", Name: "web"},
		{ID: "web", Name: "other"},
	}

	got, err := matchServiceTarget(services, "web")
	if err != nil {
		t.Fatalf("matchServiceTarget() error = %v", err)
	}
	if got.ID != "web" {
		t.Fatalf("matchServiceTarget() ID = %q, want %q", got.ID, "web")
	}
}

func TestMatchServiceTargetRejectsAmbiguousName(t *testing.T) {
	services := []client.ServiceStatus{
		{ID: "svc-1", Name: "web"},
		{ID: "svc-2", Name: "web"},
	}

	_, err := matchServiceTarget(services, "web")
	if err == nil {
		t.Fatalf("matchServiceTarget() error = nil, want ambiguity error")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("matchServiceTarget() error = %q, want ambiguity message", err.Error())
	}
}
