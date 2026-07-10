package menubar

import "fmt"

// FormatProjectTitle returns the Projects submenu title for one registered main.
// Non-empty errMsg wins over clean/branch presentation.
func FormatProjectTitle(name, branch string, clean bool, errMsg string) string {
	if errMsg != "" {
		return fmt.Sprintf("%s ⚠ Error", name)
	}
	if clean {
		return fmt.Sprintf("%s ● %s", name, branch)
	}
	return fmt.Sprintf("%s ○ %s", name, branch)
}

// FormatWorktreeTitle returns a linked worktree row title (basename + clean/dirty).
func FormatWorktreeTitle(name string, clean bool) string {
	if clean {
		return fmt.Sprintf("%s ● Clean", name)
	}
	return fmt.Sprintf("%s ○ Dirty", name)
}

// FormatProjectsEmptyLabel is shown when the wrk projects registry is empty.
func FormatProjectsEmptyLabel() string {
	return "No wrk projects"
}
