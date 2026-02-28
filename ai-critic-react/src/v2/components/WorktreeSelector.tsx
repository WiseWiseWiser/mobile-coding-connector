import { useState, useRef, useEffect } from 'react';
import type { WorktreeInfo } from '../context/WorktreeContext';
import { useWorktreeRoute } from '../hooks/useWorktreeRoute';
import { FolderIcon } from '../icons/FolderIcon';
import './WorktreeSelector.css';

interface WorktreeSelectorProps {
  worktrees: WorktreeInfo[];
  currentWorktree: WorktreeInfo | null;
  onSelectWorktree: (worktreeId: number) => void;
  disabled?: boolean;
}

export function WorktreeSelector({ 
  worktrees, 
  currentWorktree, 
  onSelectWorktree,
  disabled = false 
}: WorktreeSelectorProps) {
  const [isOpen, setIsOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const { projectName } = useWorktreeRoute();

  // Close dropdown when clicking outside
  useEffect(() => {
    if (!isOpen) return;
    
    const handleClickOutside = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setIsOpen(false);
      }
    };
    
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [isOpen]);

  // Sort worktrees: main first, then by ID
  const sortedWorktrees = [...worktrees].sort((a, b) => {
    if (a.isMain !== b.isMain) return a.isMain ? -1 : 1;
    return a.id - b.id;
  });

  const handleSelect = (worktreeId: number) => {
    onSelectWorktree(worktreeId);
    setIsOpen(false);
  };

  // Display text for current worktree
  const getDisplayText = (wt: WorktreeInfo | null) => {
    if (!wt) return 'Loading...';
    if (wt.isMain) return `${projectName} (main)`;
    if (wt.branch) return `${projectName}~${wt.id} (${wt.branch})`;
    return `${projectName}~${wt.id}`;
  };

  return (
    <div 
      ref={containerRef}
      className={`worktree-selector ${disabled ? 'disabled' : ''} ${isOpen ? 'open' : ''}`}
    >
      <button
        className="worktree-selector-trigger"
        onClick={() => !disabled && setIsOpen(!isOpen)}
        disabled={disabled}
        type="button"
      >
        <FolderIcon />
        <span className="worktree-selector-text">
          {getDisplayText(currentWorktree)}
        </span>
        <span className="worktree-selector-arrow">{isOpen ? '▲' : '▼'}</span>
      </button>
      
      {isOpen && (
        <div className="worktree-selector-dropdown">
          {sortedWorktrees.map(wt => (
            <button
              key={wt.id}
              className={`worktree-option ${wt.id === currentWorktree?.id ? 'selected' : ''}`}
              onClick={() => handleSelect(wt.id)}
              type="button"
            >
              <span className="worktree-option-icon">
                {wt.isMain ? '🏠' : '📁'}
              </span>
              <span className="worktree-option-info">
                <span className="worktree-option-name">
                  {wt.isMain ? 'Main Worktree' : `Worktree ~${wt.id}`}
                </span>
                <span className="worktree-option-branch">
                  {wt.branch} • {wt.path}
                </span>
              </span>
              {wt.id === currentWorktree?.id && (
                <span className="worktree-option-check">✓</span>
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
