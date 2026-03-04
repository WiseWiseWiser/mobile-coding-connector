/**
 * Centralized app route formatting utilities.
 *
 * All functions that build navigation paths ("/project/...", "/home/...", etc.)
 * belong here so the URL shape is defined in a single place.
 */

// ---------------------------------------------------------------------------
// Worktree helpers
// ---------------------------------------------------------------------------

const WORKTREE_SEPARATOR = '~';

export interface ParsedWorktreeRoute {
    projectName: string;
    worktreeId: number;
    fullProjectName: string;
    isRootWorktree: boolean;
}

/**
 * Parse a project name that may contain a worktree suffix.
 * Format: projectName~worktreeId   (e.g. "my-project~3")
 */
export function parseWorktreeProjectName(fullProjectName: string): ParsedWorktreeRoute {
    const separatorIndex = fullProjectName.lastIndexOf(WORKTREE_SEPARATOR);

    if (separatorIndex === -1) {
        return {
            projectName: fullProjectName,
            worktreeId: 0,
            fullProjectName,
            isRootWorktree: true,
        };
    }

    const projectName = fullProjectName.substring(0, separatorIndex);
    const worktreeIdStr = fullProjectName.substring(separatorIndex + 1);
    const worktreeId = parseInt(worktreeIdStr, 10);

    if (isNaN(worktreeId)) {
        return {
            projectName: fullProjectName,
            worktreeId: 0,
            fullProjectName,
            isRootWorktree: true,
        };
    }

    return {
        projectName,
        worktreeId,
        fullProjectName,
        isRootWorktree: worktreeId === 0,
    };
}

/**
 * Build a full project name with worktree suffix.
 * Returns plain name when worktreeId is 0 (root).
 */
export function buildWorktreeProjectName(projectName: string, worktreeId: number): string {
    if (worktreeId === 0) return projectName;
    return `${projectName}${WORKTREE_SEPARATOR}${worktreeId}`;
}

// ---------------------------------------------------------------------------
// Path builders
// ---------------------------------------------------------------------------

/**
 * Project root path:  /project/{name}
 *
 * `fullProjectName` may already contain a worktree suffix (e.g. "proj~2").
 */
export function projectPath(fullProjectName: string): string {
    return `/project/${encodeURIComponent(fullProjectName)}`;
}

/**
 * Project tab path:  /project/{name}/{tab}  or  /project/{name}/{tab}/{view}
 *
 * Falls back to  /{tab}  (or  /{tab}/{view}) when `fullProjectName` is empty.
 */
export function projectTabPath(fullProjectName: string | undefined, tab: string, view?: string): string {
    const base = fullProjectName
        ? `${projectPath(fullProjectName)}/${tab}`
        : `/${tab}`;
    if (view) return `${base}/${view}`;
    return base;
}

/**
 * Build a navigation path for a tab (and optional sub-view) within a project.
 *
 * When `fullProjectName` is provided (may include worktree suffix like "proj~2"),
 * the Home tab without a view maps to  /project/{name}/home  instead of the
 * project root, matching the app's tab layout convention.
 *
 * When no project is active, Home maps to "/" and other tabs to "/{tab}".
 */
export function buildProjectNavPath(
    fullProjectName: string | undefined,
    tab: string,
    view?: string,
): string {
    if (fullProjectName) {
        if (tab === 'home' && !view) return projectTabPath(fullProjectName, 'home');
        return projectTabPath(fullProjectName, tab, view);
    }
    if (tab === 'home' && !view) return '/';
    return projectTabPath(undefined, tab, view);
}

/**
 * Tools page path:  /project/{name}/home/tools  or  /home/tools
 *
 * Optionally appends  ?tool={toolName}  to highlight a specific tool.
 */
export function toolsPath(projectName?: string, toolName?: string): string {
    const query = toolName ? `?tool=${encodeURIComponent(toolName)}` : '';
    if (projectName) {
        return `${projectPath(projectName)}/home/tools${query}`;
    }
    return `/home/tools${query}`;
}
