import { useState, useRef, useEffect } from 'react';
import type { ProjectInfo } from '../../api/projects';
import type { WorktreeInfo } from '../../hooks/project/WorktreeContext';
import './ProjectChooser.css';

export interface ProjectChooserProps {
    projects: ProjectInfo[];
    currentProject: ProjectInfo | null;
    onProjectSelect: (project: ProjectInfo) => void;
    worktreeBranch?: string | null;
    worktrees?: WorktreeInfo[];
    currentWorktree?: WorktreeInfo | null;
    onSelectWorktree?: (worktreeId: number) => void;
}

export function ProjectChooser({
    projects,
    currentProject,
    onProjectSelect,
    worktreeBranch,
    worktrees = [],
    currentWorktree,
    onSelectWorktree,
}: ProjectChooserProps) {
    const [dropdownOpen, setDropdownOpen] = useState(false);
    const [worktreeSubmenuOpen, setWorktreeSubmenuOpen] = useState(false);
    const dropdownRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        if (!dropdownOpen) return;
        const handler = (e: MouseEvent) => {
            if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
                setDropdownOpen(false);
                setWorktreeSubmenuOpen(false);
            }
        };
        document.addEventListener('mousedown', handler);
        return () => document.removeEventListener('mousedown', handler);
    }, [dropdownOpen]);

    const closeAll = () => {
        setDropdownOpen(false);
        setWorktreeSubmenuOpen(false);
    };

    const handleProjectClick = (project: ProjectInfo) => {
        onProjectSelect(project);
        closeAll();
    };

    const handleWorktreeClick = (worktreeId: number) => {
        onSelectWorktree?.(worktreeId);
        closeAll();
    };

    const isNonMainWorktree = currentWorktree != null && !currentWorktree.isMain;
    const displayName = currentProject
        ? currentProject.name + (isNonMainWorktree ? `~${currentWorktree.id}` : '')
        : 'No Project';
    const hasWorktrees = worktrees.length > 1;

    const sortedWorktrees = [...worktrees].sort((a, b) => {
        if (a.isMain !== b.isMain) return a.isMain ? -1 : 1;
        return a.id - b.id;
    });

    return (
        <div className="project-chooser" ref={dropdownRef}>
            <div
                className="project-chooser-current"
                onClick={() => {
                    setDropdownOpen(!dropdownOpen);
                    if (dropdownOpen) setWorktreeSubmenuOpen(false);
                }}
            >
                <span className="project-chooser-name">
                    {displayName}
                    {worktreeBranch && (
                        <span className="project-chooser-branch">({worktreeBranch})</span>
                    )}
                </span>
                <span className="project-chooser-chevron">▾</span>
            </div>
            {dropdownOpen && (
                <div className="project-chooser-menu">
                    {projects.map(project => {
                        const isActive = currentProject?.id === project.id;
                        return (
                            <div key={project.id} className="project-chooser-option-wrapper">
                                <button
                                    className={`project-chooser-option${isActive ? ' project-chooser-option-active' : ''}`}
                                    onClick={() => handleProjectClick(project)}
                                >
                                    <span className="project-chooser-option-name">{project.name}</span>
                                    <span className="project-chooser-option-path">{project.dir}</span>
                                </button>
                                {isActive && hasWorktrees && (
                                    <button
                                        className={`project-chooser-wt-toggle${worktreeSubmenuOpen ? ' open' : ''}`}
                                        onClick={(e) => {
                                            e.stopPropagation();
                                            setWorktreeSubmenuOpen(!worktreeSubmenuOpen);
                                        }}
                                        title="Switch worktree"
                                    >
                                        <span className="project-chooser-wt-toggle-icon">⑂</span>
                                        <span className="project-chooser-wt-toggle-chevron">{worktreeSubmenuOpen ? '▴' : '▾'}</span>
                                    </button>
                                )}
                            </div>
                        );
                    })}
                    {worktreeSubmenuOpen && hasWorktrees && (
                        <div className="project-chooser-wt-submenu">
                            <div className="project-chooser-wt-submenu-header">Worktrees</div>
                            {sortedWorktrees.map(wt => (
                                <button
                                    key={wt.id}
                                    className={`project-chooser-wt-option${wt.id === currentWorktree?.id ? ' selected' : ''}`}
                                    onClick={() => handleWorktreeClick(wt.id)}
                                >
                                    <span className="project-chooser-wt-option-info">
                                        <span className="project-chooser-wt-option-name">
                                            {wt.isMain ? 'Main' : `~${wt.id}`}
                                        </span>
                                        <span className="project-chooser-wt-option-branch">{wt.branch}</span>
                                    </span>
                                    <span className="project-chooser-wt-option-path">{wt.path}</span>
                                    {wt.id === currentWorktree?.id && (
                                        <span className="project-chooser-wt-option-check">✓</span>
                                    )}
                                </button>
                            ))}
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}
