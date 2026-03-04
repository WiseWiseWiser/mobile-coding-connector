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
