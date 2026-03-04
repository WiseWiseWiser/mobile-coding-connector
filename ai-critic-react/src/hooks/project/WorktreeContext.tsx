import { createContext, useContext, useState, useCallback, useMemo } from 'react';

export interface WorktreeInfo {
  id: number;
  path: string;
  branch: string;
  isMain: boolean;
}

export interface WorktreeContextValue {
  // Current worktree state
  currentWorktree: WorktreeInfo | null;
  worktrees: WorktreeInfo[];
  
  // Actions
  setCurrentWorktree: (worktree: WorktreeInfo | null) => void;
  setWorktrees: (worktrees: WorktreeInfo[]) => void;
  
  // Helpers
  getWorktreeById: (id: number) => WorktreeInfo | undefined;
  getWorktreeByPath: (path: string) => WorktreeInfo | undefined;
  
  // Loading state
  loading: boolean;
  error: string | null;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
}

const WorktreeCtx = createContext<WorktreeContextValue | null>(null);

export function useWorktreeContext(): WorktreeContextValue {
  const ctx = useContext(WorktreeCtx);
  if (!ctx) throw new Error('useWorktreeContext must be used within WorktreeProvider');
  return ctx;
}

export function WorktreeProvider({ children }: { children: React.ReactNode }) {
  const [worktrees, setWorktrees] = useState<WorktreeInfo[]>([]);
  const [currentWorktree, setCurrentWorktree] = useState<WorktreeInfo | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const getWorktreeById = useCallback((id: number) => {
    return worktrees.find(w => w.id === id);
  }, [worktrees]);

  const getWorktreeByPath = useCallback((path: string) => {
    return worktrees.find(w => w.path === path);
  }, [worktrees]);

  const value = useMemo(() => ({
    currentWorktree,
    worktrees,
    setCurrentWorktree,
    setWorktrees,
    getWorktreeById,
    getWorktreeByPath,
    loading,
    error,
    setLoading,
    setError,
  }), [
    currentWorktree,
    worktrees,
    getWorktreeById,
    getWorktreeByPath,
    loading,
    error,
  ]);

  return (
    <WorktreeCtx.Provider value={value}>
      {children}
    </WorktreeCtx.Provider>
  );
}
