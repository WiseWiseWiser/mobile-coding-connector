import { useMemo, useCallback } from 'react';
import { useParams, useNavigate, useLocation } from 'react-router-dom';
import {
  parseWorktreeProjectName,
  buildWorktreeProjectName,
  projectPath,
} from '../../route/route';

export type { ParsedWorktreeRoute } from '../../route/route';
export { parseWorktreeProjectName, buildWorktreeProjectName } from '../../route/route';

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
  
  const navigateToWorktree = useCallback((worktreeId: number) => {
    const newFullName = buildWorktreeProjectName(parsed.projectName, worktreeId);
    const currentPath = location.pathname;
    
    const pathParts = currentPath.split('/');
    const projectIndex = pathParts.findIndex(p => p === 'project');
    if (projectIndex !== -1 && pathParts[projectIndex + 1]) {
      pathParts[projectIndex + 1] = newFullName;
      const newPath = pathParts.join('/');
      navigate(newPath, { replace: true });
    }
  }, [parsed.projectName, location.pathname, navigate]);
  
  const buildWorktreePath = useCallback((worktreeId: number, view?: string) => {
    const fullName = buildWorktreeProjectName(parsed.projectName, worktreeId);
    const path = projectPath(fullName);
    if (view) return `${path}/${view}`;
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
