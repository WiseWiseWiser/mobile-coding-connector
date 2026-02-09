import { useState, useRef, useEffect } from 'react';
import type { ProjectInfo } from '../../api/projects';

export interface ProjectDropdownProps {
    projects: ProjectInfo[];
    currentProject: ProjectInfo | null;
    onProjectSelect: (project: ProjectInfo) => void;
}

export function ProjectDropdown({ projects, currentProject, onProjectSelect }: ProjectDropdownProps) {
    const [dropdownOpen, setDropdownOpen] = useState(false);
    const dropdownRef = useRef<HTMLDivElement>(null);

    // Close dropdown on outside click
    useEffect(() => {
        if (!dropdownOpen) return;
        const handler = (e: MouseEvent) => {
            if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
                setDropdownOpen(false);
            }
        };
        document.addEventListener('mousedown', handler);
        return () => document.removeEventListener('mousedown', handler);
    }, [dropdownOpen]);

    const handleProjectClick = (project: ProjectInfo) => {
        onProjectSelect(project);
        setDropdownOpen(false);
    };

    return (
        <div className="mcc-project-dropdown" ref={dropdownRef}>
            <div
                className="mcc-project-current"
                onClick={() => setDropdownOpen(!dropdownOpen)}
            >
                <span className="mcc-project-name">
                    {currentProject ? currentProject.name : 'No Project'}
                </span>
                <span className="mcc-project-chevron">▾</span>
            </div>
            {dropdownOpen && (
                <div className="mcc-project-menu">
                    {projects.map(project => (
                        <button
                            key={project.id}
                            className={`mcc-project-option${currentProject?.id === project.id ? ' mcc-project-option-active' : ''}`}
                            onClick={() => handleProjectClick(project)}
                        >
                            <span className="mcc-project-option-name">{project.name}</span>
                            <span className="mcc-project-option-path">{project.dir}</span>
                        </button>
                    ))}
                </div>
            )}
        </div>
    );
}