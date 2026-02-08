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

// rcPatchOptions holds the options for patching shell RC files.
type rcPatchOptions struct {
	ExtraPaths []string
	PS1        string
}

// buildPatchBlock builds the patch block for the given options.
func buildPatchBlock(opts rcPatchOptions) string {
	var lines []string
	if len(opts.ExtraPaths) > 0 {
		lines = append(lines, fmt.Sprintf("export PATH=$PATH:%s", strings.Join(opts.ExtraPaths, ":")))
	}
	if opts.PS1 != "" {
		lines = append(lines, fmt.Sprintf("export PS1=%s", ShellQuote(opts.PS1)))
	}
	if len(lines) == 0 {
		return ""
	}
	return fmt.Sprintf("%s\n%s\n%s\n",
		patchBeginMarker,
		strings.Join(lines, "\n"),
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
// ai-critic patch with exactly the given options (extra paths, PS1, etc.).
// It first removes any existing patch, then appends the new one.
// If opts produces no patch lines, it only removes the existing patch.
func patchRCFiles(opts rcPatchOptions) error {
	files := rcFiles()
	if len(files) == 0 {
		return nil
	}

	patchBlock := buildPatchBlock(opts)

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
