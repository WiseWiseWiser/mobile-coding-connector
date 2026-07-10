package menubar

// ProjectTitleParts is the left/right split for a Projects submenu title.
// Leading is the basename only; Trailing is the decoration (branch / error).
type ProjectTitleParts struct {
	Leading  string
	Trailing string
}

// WorktreeTitleParts is the left/right split for a linked worktree row.
type WorktreeTitleParts struct {
	Leading  string
	Trailing string
}

// FormatProjectTitleParts splits a project row into Leading (name) and Trailing decoration.
// Non-empty errMsg wins over clean/branch presentation.
func FormatProjectTitleParts(name, branch string, clean bool, errMsg string) ProjectTitleParts {
	parts := ProjectTitleParts{Leading: name}
	if errMsg != "" {
		parts.Trailing = "⚠ Error"
		return parts
	}
	if clean {
		parts.Trailing = "● " + branch
		return parts
	}
	parts.Trailing = "○ " + branch
	return parts
}

// FormatWorktreeTitleParts splits a worktree row into Leading (name) and Trailing clean/dirty.
func FormatWorktreeTitleParts(name string, clean bool) WorktreeTitleParts {
	parts := WorktreeTitleParts{Leading: name}
	if clean {
		parts.Trailing = "● Clean"
		return parts
	}
	parts.Trailing = "○ Dirty"
	return parts
}

// FormatProjectTitle returns the legacy single-string Projects submenu title.
// Composed as Leading + "  " + Trailing (double space).
func FormatProjectTitle(name, branch string, clean bool, errMsg string) string {
	p := FormatProjectTitleParts(name, branch, clean, errMsg)
	return p.Leading + "  " + p.Trailing
}

// FormatWorktreeTitle returns a legacy single-string linked worktree row title.
// Composed as Leading + "  " + Trailing (double space).
func FormatWorktreeTitle(name string, clean bool) string {
	p := FormatWorktreeTitleParts(name, clean)
	return p.Leading + "  " + p.Trailing
}

// FormatProjectsEmptyLabel is shown when the wrk projects registry is empty.
func FormatProjectsEmptyLabel() string {
	return "No wrk projects"
}

// FormatProjectsLoadingLabel is shown while the projects list is in flight and empty.
// Uses unicode ellipsis U+2026, not three ASCII periods.
func FormatProjectsLoadingLabel() string {
	return "Loading…"
}

// FormatProjectsLoadFailedLabel is shown when the list failed and there are no rows.
func FormatProjectsLoadFailedLabel() string {
	return "Failed to load projects"
}

// FormatProjectsListStatusLabel picks the disabled placeholder for the empty-area
// of the Projects menu. When count > 0, returns "" (show project menus).
func FormatProjectsListStatusLabel(loading bool, count int, err string) string {
	if count > 0 {
		return ""
	}
	if loading {
		return FormatProjectsLoadingLabel()
	}
	if err != "" {
		return FormatProjectsLoadFailedLabel()
	}
	return FormatProjectsEmptyLabel()
}

// ProjectsListState is the pure in-memory model for stale-while-revalidate list refresh.
// Projects is a simplified token list (paths/names) for reducers and tests.
type ProjectsListState struct {
	Projects []string
	Loading  bool
	Error    string
}

// ApplyProjectsRefreshStart marks loading without clearing items or inventing an error.
func ApplyProjectsRefreshStart(s ProjectsListState) ProjectsListState {
	s.Loading = true
	return s
}

// ApplyProjectsRefreshSuccess replaces the list, clears error, and ends loading.
func ApplyProjectsRefreshSuccess(s ProjectsListState, list []string) ProjectsListState {
	s.Projects = append([]string(nil), list...)
	s.Error = ""
	s.Loading = false
	return s
}

// ApplyProjectsRefreshFailure keeps items, records err, and ends loading.
func ApplyProjectsRefreshFailure(s ProjectsListState, err string) ProjectsListState {
	s.Error = err
	s.Loading = false
	return s
}
