package daemon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetCurrentExecutablePath(t *testing.T) {
	// Test the getCurrentExecutablePath function
	path, err := getCurrentExecutablePath()
	if err != nil {
		t.Fatalf("getCurrentExecutablePath failed: %v", err)
	}
	t.Logf("Current executable path: %s", path)

	// Test os.Executable
	exePath, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable failed: %v", err)
	}
	t.Logf("os.Executable: %s", exePath)

	// Test filepath.EvalSymlinks
	realPath, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		t.Logf("filepath.EvalSymlinks error: %v", err)
	} else {
		t.Logf("filepath.EvalSymlinks: %s", realPath)
	}

	// Check if file exists
	if _, err := os.Stat(path); err != nil {
		t.Logf("File does NOT exist: %s", path)
	} else {
		t.Logf("File EXISTS: %s", path)
	}

	// Check /proc/self/exe
	if _, err := os.Stat("/proc/self/exe"); err == nil {
		procExe, _ := os.Readlink("/proc/self/exe")
		t.Logf("/proc/self/exe: %s", procExe)
	}
}

func TestParseBinVersion(t *testing.T) {
	tests := []struct {
		input    string
		wantBase string
		wantVer  int
	}{
		{"ai-critic-server", "ai-critic-server", 0},
		{"ai-critic-server-v1", "ai-critic-server", 1},
		{"ai-critic-server-v5", "ai-critic-server", 5},
		{"/path/to/ai-critic-server-v10", "/path/to/ai-critic-server", 10},
		{"ai-critic-server-linux-amd64", "ai-critic-server-linux-amd64", 0},
		{"ai-critic-server-linux-amd64-v2", "ai-critic-server-linux-amd64", 2},
	}

	for _, tt := range tests {
		base, ver := ParseBinVersion(tt.input)
		if base != tt.wantBase || ver != tt.wantVer {
			t.Errorf("ParseBinVersion(%q) = (%q, %d), want (%q, %d)",
				tt.input, base, ver, tt.wantBase, tt.wantVer)
		}
	}
}

func TestFindNewerBinary(t *testing.T) {
	// Test with current executable
	currentPath, _ := getCurrentExecutablePath()
	t.Logf("Testing FindNewerBinary with current path: %s", currentPath)

	newer := FindNewerBinary(currentPath)
	if newer != "" {
		t.Logf("Found newer binary: %s", newer)
	} else {
		t.Logf("No newer binary found")
	}

	// Also test with the problematic path
	problemPath := "/root/ai-critic-server-v5"
	t.Logf("Testing FindNewerBinary with problem path: %s", problemPath)
	newer2 := FindNewerBinary(problemPath)
	if newer2 != "" {
		t.Logf("Found newer binary: %s", newer2)
	} else {
		t.Logf("No newer binary found for problem path")
	}
}
