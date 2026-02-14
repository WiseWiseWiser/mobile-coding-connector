package checkpoint

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/xhd2015/lifelog-private/ai-critic/server/gitrunner"
)

// FileDiff represents the unified diff for a single file.
type FileDiff struct {
	Path   string     `json:"path"`
	Status string     `json:"status"`
	Hunks  []DiffHunk `json:"hunks"`
}

// DiffHunk represents a hunk in a unified diff.
type DiffHunk struct {
	OldStart int        `json:"old_start"`
	OldLines int        `json:"old_lines"`
	NewStart int        `json:"new_start"`
	NewLines int        `json:"new_lines"`
	Lines    []DiffLine `json:"lines"`
}

// DiffLine is a single line in a diff hunk.
type DiffLine struct {
	Type    string `json:"type"` // "context", "add", "delete"
	Content string `json:"content"`
	OldNum  int    `json:"old_num,omitempty"`
	NewNum  int    `json:"new_num,omitempty"`
}

// GetCheckpointDiff computes diffs for all files in a checkpoint.
// Uses original/ directory for the "before" content (git HEAD at checkpoint time).
func GetCheckpointDiff(projectName string, id int) ([]FileDiff, error) {
	mu.RLock()
	defer mu.RUnlock()

	list, err := loadCheckpoints(projectName)
	if err != nil {
		return nil, err
	}

	// Find the checkpoint
	var cp *Checkpoint
	for i := range list {
		if list[i].ID == id {
			cp = &list[i]
			break
		}
	}
	if cp == nil {
		return nil, fmt.Errorf("checkpoint %d not found", id)
	}

	diffs := make([]FileDiff, 0, len(cp.Files))
	for _, f := range cp.Files {
		var oldContent, newContent string

		// Get original content (git HEAD at checkpoint time)
		if f.Status == "modified" || f.Status == "deleted" {
			content, err := getOriginalContent(cp.DirPath, f.Path)
			if err == nil {
				oldContent = content
			}
		}
		// For "added" files, oldContent stays empty

		// Get new content
		if f.Status != "deleted" {
			content, err := getFileContent(cp.DirPath, f.Path)
			if err != nil {
				// Skip files we can't read
				continue
			}
			newContent = content
		}
		// For "deleted" files, newContent stays empty

		hunks := computeUnifiedDiff(oldContent, newContent)
		diffs = append(diffs, FileDiff{
			Path:   f.Path,
			Status: f.Status,
			Hunks:  hunks,
		})
	}

	return diffs, nil
}

// GetSingleFileDiff computes diff for a single file.
func GetSingleFileDiff(projectDir, filePath string) (*FileDiff, error) {
	// Get the status of this specific file
	status, err := gitFileStatus(projectDir, filePath)
	if err != nil {
		return nil, err
	}

	if status == "" {
		// File is not changed, return empty diff
		return &FileDiff{
			Path:   filePath,
			Status: "unchanged",
			Hunks:  nil,
		}, nil
	}

	var oldContent, newContent string

	// Get git HEAD content for modified/deleted files
	if status == "modified" || status == "deleted" {
		content, err := gitFileContent(projectDir, filePath)
		if err == nil {
			oldContent = content
		}
	}

	// Get current disk content for added/modified files
	if status != "deleted" {
		content, err := readFileContent(projectDir, filePath)
		if err == nil {
			newContent = content
		}
	}

	hunks := computeUnifiedDiff(oldContent, newContent)
	return &FileDiff{
		Path:   filePath,
		Status: status,
		Hunks:  hunks,
	}, nil
}

// gitFileStatus returns the status of a single file.
func gitFileStatus(projectDir, filePath string) (string, error) {
	out, err := gitrunner.Diff("--name-status", "HEAD", "--", filePath).Dir(projectDir).Output()
	if err != nil {
		// No changes in this file
		return "", nil
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return "", nil
	}

	parts := strings.SplitN(lines[0], "\t", 2)
	if len(parts) != 2 {
		return "", nil
	}

	return parseGitStatus(parts[0]), nil
}

// GetCurrentDiff computes diffs for current working tree changes against git HEAD.
func GetCurrentDiff(projectDir string) ([]FileDiff, error) {
	// Get list of changed files
	changes, err := gitChangedFiles(projectDir)
	if err != nil {
		return nil, err
	}

	diffs := make([]FileDiff, 0, len(changes))
	for _, f := range changes {
		var oldContent, newContent string

		// Get git HEAD content for modified/deleted files
		if f.Status == "modified" || f.Status == "deleted" {
			content, err := gitFileContent(projectDir, f.Path)
			if err == nil {
				oldContent = content
			}
		}

		// Get current disk content for added/modified files
		if f.Status != "deleted" {
			content, err := readFileContent(projectDir, f.Path)
			if err == nil {
				newContent = content
			}
		}

		hunks := computeUnifiedDiff(oldContent, newContent)
		diffs = append(diffs, FileDiff{
			Path:   f.Path,
			Status: f.Status,
			Hunks:  hunks,
		})
	}

	return diffs, nil
}

// computeUnifiedDiff computes a simple unified diff between old and new content.
func computeUnifiedDiff(oldText, newText string) []DiffHunk {
	oldLines := splitLines(oldText)
	newLines := splitLines(newText)

	// Simple LCS-based diff
	editScript := computeEditScript(oldLines, newLines)

	// Group edits into hunks with context
	const contextLines = 3
	return groupIntoHunks(editScript, oldLines, newLines, contextLines)
}

type editOp struct {
	kind    byte // 'E' equal, 'D' delete, 'I' insert
	oldIdx  int
	newIdx  int
	content string
}

func computeEditScript(oldLines, newLines []string) []editOp {
	m, n := len(oldLines), len(newLines)

	// Build LCS table
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if oldLines[i-1] == newLines[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	// Backtrace to produce edit script
	var ops []editOp
	i, j := m, n
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && oldLines[i-1] == newLines[j-1] {
			ops = append(ops, editOp{'E', i - 1, j - 1, oldLines[i-1]})
			i--
			j--
		} else if i > 0 && (j == 0 || dp[i-1][j] >= dp[i][j-1]) {
			ops = append(ops, editOp{'D', i - 1, -1, oldLines[i-1]})
			i--
		} else {
			ops = append(ops, editOp{'I', -1, j - 1, newLines[j-1]})
			j--
		}
	}

	// Reverse to get forward order
	for l, r := 0, len(ops)-1; l < r; l, r = l+1, r-1 {
		ops[l], ops[r] = ops[r], ops[l]
	}
	return ops
}

func groupIntoHunks(ops []editOp, oldLines, newLines []string, contextLines int) []DiffHunk {
	if len(ops) == 0 {
		return nil
	}

	// Find ranges of changes
	type changeRange struct {
		start, end int // indices in ops
	}
	var changes []changeRange
	for i, op := range ops {
		if op.kind != 'E' {
			if len(changes) == 0 || i > changes[len(changes)-1].end+contextLines*2 {
				changes = append(changes, changeRange{i, i})
			} else {
				changes[len(changes)-1].end = i
			}
		}
	}

	var hunks []DiffHunk
	for _, cr := range changes {
		// Expand range with context
		start := cr.start - contextLines
		if start < 0 {
			start = 0
		}
		end := cr.end + contextLines + 1
		if end > len(ops) {
			end = len(ops)
		}

		var lines []DiffLine
		oldStart, newStart := 0, 0
		oldCount, newCount := 0, 0

		// Calculate starting line numbers
		for i := 0; i < start; i++ {
			switch ops[i].kind {
			case 'E':
				oldStart++
				newStart++
			case 'D':
				oldStart++
			case 'I':
				newStart++
			}
		}

		hunkOldStart := oldStart + 1
		hunkNewStart := newStart + 1

		for i := start; i < end; i++ {
			op := ops[i]
			switch op.kind {
			case 'E':
				oldStart++
				newStart++
				oldCount++
				newCount++
				lines = append(lines, DiffLine{Type: "context", Content: op.content, OldNum: oldStart, NewNum: newStart})
			case 'D':
				oldStart++
				oldCount++
				lines = append(lines, DiffLine{Type: "delete", Content: op.content, OldNum: oldStart})
			case 'I':
				newStart++
				newCount++
				lines = append(lines, DiffLine{Type: "add", Content: op.content, NewNum: newStart})
			}
		}

		hunks = append(hunks, DiffHunk{
			OldStart: hunkOldStart,
			OldLines: oldCount,
			NewStart: hunkNewStart,
			NewLines: newCount,
			Lines:    lines,
		})
	}

	return hunks
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	// Remove trailing empty line from trailing newline
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// handleCheckpointDiff handles GET /api/checkpoints/{id}/diff
func handleCheckpointDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	project := r.URL.Query().Get("project")
	if project == "" {
		respondErr(w, http.StatusBadRequest, "project is required")
		return
	}

	// Parse: /api/checkpoints/{id}/diff
	path := strings.TrimPrefix(r.URL.Path, "/api/checkpoints/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 || parts[1] != "diff" {
		respondErr(w, http.StatusBadRequest, "invalid path")
		return
	}

	id, err := strconv.Atoi(parts[0])
	if err != nil {
		respondErr(w, http.StatusBadRequest, "invalid checkpoint id")
		return
	}

	diffs, err := GetCheckpointDiff(project, id)
	if err != nil {
		respondErr(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, diffs)
}
