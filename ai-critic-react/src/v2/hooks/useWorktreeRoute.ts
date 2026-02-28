import { useMemo, useCallback } from 'react';
import { useParams, useNavigate, useLocation } from 'react-router-dom';

export interface ParsedWorktreeRoute {
  /** The project name (without worktree suffix) */
  projectName: string;
  /** The worktree ID (0 for root worktree) */
  worktreeId: number;
  /** The full project name with worktree suffix for URL */
  fullProjectName: string;
  /** Whether this is the root worktree */
  isRootWorktree: boolean;
}

const WORKTREE_SEPARATOR = '~';

/**
 * Parse a project name that may contain a worktree suffix.
 * Format: projectName~worktreeId
 * Example: my-project~3
 */
export function parseWorktreeProjectName(fullProjectName: string): ParsedWorktreeRoute {
  const separatorIndex = fullProjectName.lastIndexOf(WORKTREE_SEPARATOR);
  
  if (separatorIndex === -1) {
    // No worktree suffix - this is the root worktree
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
    // Invalid worktree ID - treat as root
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
 */
export function buildWorktreeProjectName(projectName: string, worktreeId: number): string {
  if (worktreeId === 0) {
    return projectName;
  }
  return `${projectName}${WORKTREE_SEPARATOR}${worktreeId}`;
}

/**
 * React hook for working with worktree routes.
 */
export function useWorktreeRoute() {
  const params = useParams<{ projectName?: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  
  const parsed = useMemo(() => {
    const fullProjectName = params.projectName || '';
    return parseWorktreeProjectName(fullProjectName);
  }, [params.projectName]);
  
  /**
   * Navigate to a different worktree while keeping the same view/tab.
   */
  const navigateToWorktree = useCallback((worktreeId: number) => {
    const newFullName = buildWorktreeProjectName(parsed.projectName, worktreeId);
    const currentPath = location.pathname;
    
    // Replace the project name in the current path
    const pathParts = currentPath.split('/');
    const projectIndex = pathParts.findIndex(p => p === 'project');
    if (projectIndex !== -1 && pathParts[projectIndex + 1]) {
      pathParts[projectIndex + 1] = newFullName;
      const newPath = pathParts.join('/');
      navigate(newPath, { replace: true });
    }
  }, [parsed.projectName, location.pathname, navigate]);
  
  /**
   * Build a URL path for a specific worktree and view.
   */
  const buildWorktreePath = useCallback((worktreeId: number, view?: string) => {
    const fullName = buildWorktreeProjectName(parsed.projectName, worktreeId);
    let path = `/project/${fullName}`;
    if (view) {
      path += `/${view}`;
    }
    return path;
  }, [parsed.projectName]);
  
  return {
    ...parsed,
    navigateToWorktree,
    buildWorktreePath,
    rawProjectName: params.projectName,
  };
}

export default useWorktreeRoute;
