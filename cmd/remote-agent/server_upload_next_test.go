package main

import (
	"testing"

	"github.com/xhd2015/ai-critic/client"
)

func TestNextBinaryTargetFromRemoteEntriesUsesHighestExistingVersion(t *testing.T) {
	entries := []client.BrowseEntry{
		{Name: "ai-critic-server-v1"},
		{Name: "ai-critic-server-v2"},
		{Name: "ai-critic-server-v10"},
		{Name: "ai-critic-server-v11"},
		{Name: "ai-critic-server-v12"},
		{Name: "ai-critic-server-v13"},
		{Name: "ai-critic-server-v15"},
	}

	target, err := nextBinaryTargetFromRemoteEntries("/opt/ai-critic/ai-critic-server-v13", entries)
	if err != nil {
		t.Fatalf("nextBinaryTargetFromRemoteEntries() error = %v", err)
	}
	if target.BinaryPath != "/opt/ai-critic/ai-critic-server-v16" {
		t.Fatalf("target.BinaryPath = %q, want /opt/ai-critic/ai-critic-server-v16", target.BinaryPath)
	}
	if target.PreviousHighestVersion != 15 {
		t.Fatalf("target.PreviousHighestVersion = %d, want 15", target.PreviousHighestVersion)
	}
}

func TestNextBinaryTargetFromRemoteEntriesIgnoresOtherBases(t *testing.T) {
	entries := []client.BrowseEntry{
		{Name: "ai-critic-server-v1"},
		{Name: "ai-critic-server-v2"},
		{Name: "other-server-v99"},
	}

	target, err := nextBinaryTargetFromRemoteEntries("/opt/ai-critic/ai-critic-server-v2", entries)
	if err != nil {
		t.Fatalf("nextBinaryTargetFromRemoteEntries() error = %v", err)
	}
	if target.BinaryName != "ai-critic-server-v3" {
		t.Fatalf("target.BinaryName = %q, want ai-critic-server-v3", target.BinaryName)
	}
}
