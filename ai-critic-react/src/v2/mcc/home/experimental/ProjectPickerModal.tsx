import { useState, useMemo } from 'react';
import type { ProjectInfo } from '../../../../api/projects';
import { buildWorktreeProjectName } from '../../../../route/route';
import './ProjectPickerModal.css';

interface ProjectPickerModalProps {
    projects: ProjectInfo[];
    onSelect: (fullProjectName: string) => void;
    onClose: () => void;
}

export function ProjectPickerModal({ projects, onSelect, onClose }: ProjectPickerModalProps) {
    const [filter, setFilter] = useState('');
    const [expandedId, setExpandedId] = useState<string | null>(null);

    const filtered = useMemo(() => {
        if (!filter.trim()) return projects;
        const q = filter.toLowerCase();
        return projects.filter(p => p.name.toLowerCase().includes(q));
    }, [projects, filter]);

    return (
        <div className="project-picker-overlay" onClick={onClose}>
            <div className="project-picker-modal" onClick={e => e.stopPropagation()}>
                <div className="project-picker-header">
                    <h3>Select Project</h3>
                    <button className="project-picker-close" onClick={onClose}>&times;</button>
                </div>

                <input
                    className="project-picker-search"
                    type="text"
                    placeholder="Filter projects..."
                    value={filter}
                    onChange={e => setFilter(e.target.value)}
                    autoFocus
                />

                <div className="project-picker-list">
                    {filtered.length === 0 ? (
                        <div className="project-picker-empty">No projects found</div>
                    ) : (
                        filtered.map(project => {
                            const worktreeEntries = project.worktrees ? Object.entries(project.worktrees) : [];
                            const hasWorktrees = worktreeEntries.length > 0;
                            const isExpanded = expandedId === project.id;

                            return (
                                <div key={project.id} className="project-picker-item-group">
                                    <div
                                        className="project-picker-item"
                                        onClick={() => {
                                            if (hasWorktrees && !isExpanded) {
                                                setExpandedId(project.id);
                                            } else {
                                                onSelect(project.name);
                                            }
                                        }}
                                    >
                                        <div className="project-picker-item-info">
                                            <span className="project-picker-item-name">{project.name}</span>
                                            {project.dir && (
                                                <span className="project-picker-item-dir">{project.dir}</span>
                                            )}
                                        </div>
                                        {hasWorktrees && (
                                            <span
                                                className={`project-picker-expand ${isExpanded ? 'expanded' : ''}`}
                                                onClick={e => {
                                                    e.stopPropagation();
                                                    setExpandedId(isExpanded ? null : project.id);
                                                }}
                                            >
                                                {isExpanded ? '▼' : '▶'}
                                            </span>
                                        )}
                                    </div>
                                    {isExpanded && hasWorktrees && (
                                        <div className="project-picker-worktrees">
                                            <div
                                                className="project-picker-worktree"
                                                onClick={() => onSelect(project.name)}
                                            >
                                                <span className="project-picker-wt-branch">main (default)</span>
                                            </div>
                                            {worktreeEntries.map(([wtId, wt]) => (
                                                <div
                                                    key={wtId}
                                                    className="project-picker-worktree"
                                                    onClick={() => onSelect(buildWorktreeProjectName(project.name, Number(wtId)))}
                                                >
                                                    <span className="project-picker-wt-branch">{wt.branch}</span>
                                                    <span className="project-picker-wt-path">{wt.path}</span>
                                                </div>
                                            ))}
                                        </div>
                                    )}
                                </div>
                            );
                        })
                    )}
                </div>
            </div>
        </div>
    );
}
