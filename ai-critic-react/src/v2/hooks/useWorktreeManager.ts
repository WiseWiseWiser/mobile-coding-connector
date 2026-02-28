import { useState, useCallback } from 'react';
import type { ProjectInfo, WorktreeConfig } from '../../api/projects';
import type { WorktreeInfo } from '../context/WorktreeContext';
import { listWorktrees as apiListWorktrees } from '../../api/review';
import { updateProject } from '../../api/projects';

export interface UseWorktreeManagerResult {
  worktrees: WorktreeInfo[];
  loading: boolean;
  error: string | null;
  loadWorktrees: (project: ProjectInfo) => Promise<void>;
  refreshWorktrees: () => Promise<void>;
  assignWorktreeId: (path: string, branch: string) => Promise<number>;
}

/**
 * Hook to manage worktree loading and ID assignment.
 */
export function useWorktreeManager(): UseWorktreeManagerResult {
  const [worktrees, setWorktrees] = useState<WorktreeInfo[]>([]);
  const [currentProject, setCurrentProject] = useState<ProjectInfo | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  /**
   * Load worktrees for a project and sync with stored config.
   */
  const loadWorktrees = useCallback(async (project: ProjectInfo) => {
    setLoading(true);
    setError(null);
    setCurrentProject(project);
    
    try {
      // Fetch worktrees from git
      const gitWorktrees = await apiListWorktrees(project.dir);
      
      // Get or initialize worktree config
      const worktreeConfig: WorktreeConfig = project.worktrees || {};
      
      // Map worktrees and assign IDs
      const mappedWorktrees: WorktreeInfo[] = [];
      
      // Find the main worktree first
      const mainWorktree = gitWorktrees.find(wt => wt.isMain);
      if (mainWorktree) {
        mappedWorktrees.push({
          id: 0,
          path: mainWorktree.path,
          branch: mainWorktree.branch,
          isMain: true,
        });
      }
      
      // Process remaining worktrees
      for (const wt of gitWorktrees) {
        if (wt.isMain) continue; // Already added
        
        // Try to find existing ID in config
        let id: number | undefined;
        for (const [key, value] of Object.entries(worktreeConfig)) {
          if (value.path === wt.path) {
            id = parseInt(key, 10);
            break;
          }
        }
        
        // If no ID found, assign new one
        if (id === undefined) {
          // Get all existing IDs
          const existingIds = [
            0, // Root worktree
            ...mappedWorktrees.map(w => w.id),
            ...Object.keys(worktreeConfig).map(k => parseInt(k, 10)),
          ];
          id = existingIds.length > 0 ? Math.max(...existingIds) + 1 : 1;
          
          // Save to config (but don't await - it's okay if it fails)
          saveWorktreeConfig(project, id, wt.path, wt.branch);
        }
        
        mappedWorktrees.push({
          id,
          path: wt.path,
          branch: wt.branch,
          isMain: false,
        });
      }
      
      setWorktrees(mappedWorktrees);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load worktrees');
    } finally {
      setLoading(false);
    }
  }, []);
  
  /**
   * Save worktree config to project.
   */
  const saveWorktreeConfig = async (project: ProjectInfo, id: number, path: string, branch: string) => {
    try {
      const worktrees: WorktreeConfig = {
        ...project.worktrees,
        [id]: { path, branch },
      };
      await updateProject(project.id, { worktrees });
    } catch (err) {
      console.error('Failed to save worktree config:', err);
    }
  };

  /**
   * Refresh worktrees (reload from server).
   */
  const refreshWorktrees = useCallback(async () => {
    if (currentProject) {
      await loadWorktrees(currentProject);
    }
  }, [currentProject, loadWorktrees]);

  /**
   * Assign a new worktree ID.
   */
  const assignWorktreeId = useCallback(async (path: string, branch: string): Promise<number> => {
    if (!currentProject) throw new Error('No current project');
    
    const existingIds = worktrees.map(w => w.id);
    const newId = existingIds.length > 0 ? Math.max(...existingIds) + 1 : 1;
    
    // Save to project config
    await saveWorktreeConfig(currentProject, newId, path, branch);
    
    // Add to local state
    const newWorktree: WorktreeInfo = {
      id: newId,
      path,
      branch,
      isMain: false,
    };
    setWorktrees(prev => [...prev, newWorktree]);
    
    return newId;
  }, [currentProject, worktrees]);

  return {
    worktrees,
    loading,
    error,
    loadWorktrees,
    refreshWorktrees,
    assignWorktreeId,
  };
}
