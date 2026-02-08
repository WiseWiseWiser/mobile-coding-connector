package terminal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	patchBeginMarker = "# === ai-critic PATH PATCH begin ==="
	patchEndMarker   = "# === ai-critic PATH PATCH end ==="
)

// rcFiles returns the list of shell RC files that exist on the system.
func rcFiles() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	candidates := []string{
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".bash_profile"),
		filepath.Join(home, ".zshrc"),
	}

	var found []string
	for _, f := range candidates {
		if _, err := os.Stat(f); err == nil {
			found = append(found, f)
		}
	}
	return found
}

// buildPatchBlock builds the PATH patch block for the given extra paths.
func buildPatchBlock(extraPaths []string) string {
	if len(extraPaths) == 0 {
		return ""
	}
	return fmt.Sprintf("%s\nexport PATH=$PATH:%s\n%s\n",
		patchBeginMarker,
		strings.Join(extraPaths, ":"),
		patchEndMarker,
	)
}

// removePatch removes the existing ai-critic PATH patch from content.
func removePatch(content string) string {
	beginIdx := strings.Index(content, patchBeginMarker)
	if beginIdx < 0 {
		return content
	}
	endIdx := strings.Index(content, patchEndMarker)
	if endIdx < 0 {
		return content
	}
	endIdx += len(patchEndMarker)
	// Also consume a trailing newline if present
	if endIdx < len(content) && content[endIdx] == '\n' {
		endIdx++
	}
	return content[:beginIdx] + content[endIdx:]
}

// patchRCFiles ensures that all existing shell RC files contain the
// ai-critic PATH patch with exactly the given extra paths.
// It first removes any existing patch, then appends the new one.
// If extraPaths is empty, it only removes the existing patch.
func patchRCFiles(extraPaths []string) error {
	files := rcFiles()
	if len(files) == 0 {
		return nil
	}

	patchBlock := buildPatchBlock(extraPaths)

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read %s: %w", f, err)
		}

		content := removePatch(string(data))

		if patchBlock != "" {
			// Ensure there's a newline before the patch
			if len(content) > 0 && content[len(content)-1] != '\n' {
				content += "\n"
			}
			content += patchBlock
		}

		if err := os.WriteFile(f, []byte(content), 0644); err != nil {
			return fmt.Errorf("write %s: %w", f, err)
		}
	}
	return nil
}
