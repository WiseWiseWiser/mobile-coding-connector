import { Outlet, useNavigate, useOutletContext } from 'react-router-dom';
import { deleteProject as apiDeleteProject } from '../api/projects';
import type { ProjectInfo } from '../api/projects';
import { useV2Context } from './V2Context';

// Re-export sub-views for route registration
export { DiagnoseView } from './DiagnoseView';
export { GitSettings, CloneRepoView } from './GitSettings';

interface HomeOutletContext {
    onSelectProject: (project: ProjectInfo) => void;
}

interface WorkspaceListViewProps {
    onSelectProject?: (project: ProjectInfo) => void;
}

export function WorkspaceListView({ onSelectProject: propOnSelectProject }: WorkspaceListViewProps) {
    const navigate = useNavigate();
    const { projectsList, projectsLoading, fetchProjects, currentProject } = useV2Context();
    
    // Get onSelectProject from outlet context if not provided as prop
    const outletContext = useOutletContext<HomeOutletContext | null>();
    const onSelectProject = propOnSelectProject ?? outletContext?.onSelectProject ?? (() => {});

    const handleRemoveProject = async (id: string) => {
        try {
            await apiDeleteProject(id);
            fetchProjects();
        } catch {
            // ignore
        }
    };

    return (
        <div className="mcc-workspace-list">
            <div className="mcc-section-header">
                <h2>Your Projects</h2>
            </div>
            {projectsLoading ? (
                <div className="mcc-ports-empty">Loading projects...</div>
            ) : projectsList.length === 0 ? (
                <div className="mcc-ports-empty">No projects yet. Clone a repository from Git Settings.</div>
            ) : (
                <div className="mcc-workspace-cards">
                    {projectsList.map(project => (
                        <ProjectCard
                            key={project.id}
                            project={project}
                            isActive={project.name === currentProject?.name}
                            onSelect={() => onSelectProject(project)}
                            onRemove={() => handleRemoveProject(project.id)}
                        />
                    ))}
                </div>
            )}
            <button className="mcc-new-workspace-btn" onClick={() => navigate('clone-repo')}>
                <PlusIcon />
                <span>Clone Repository</span>
            </button>
            <button className="mcc-git-settings-btn" onClick={() => navigate('git-settings')}>
                <GitIcon />
                <span>Git Settings</span>
            </button>
            <button className="mcc-diagnose-btn" onClick={() => navigate('diagnose')}>
                <DiagnoseIcon />
                <span>System Diagnostics</span>
            </button>
        </div>
    );
}

// Project Card
interface ProjectCardProps {
    project: ProjectInfo;
    isActive: boolean;
    onSelect: () => void;
    onRemove: () => void;
}

function ProjectCard({ project, isActive, onSelect, onRemove }: ProjectCardProps) {
    const createdDate = new Date(project.created_at).toLocaleDateString();

    return (
        <div className={`mcc-workspace-card mcc-workspace-card-clickable${isActive ? ' mcc-workspace-card-active' : ''}`} onClick={onSelect}>
            <div className="mcc-workspace-card-header">
                <span className="mcc-workspace-name">{project.name}</span>
                {isActive && <span className="mcc-workspace-active-badge">Working on</span>}
            </div>
            <div className="mcc-workspace-card-meta">
                <span>{project.dir}</span>
            </div>
            <div className="mcc-workspace-card-meta" style={{ marginTop: 4 }}>
                <span>{project.repo_url}</span>
                <span>{createdDate}</span>
            </div>
            <div className="mcc-port-actions" style={{ marginTop: 8 }}>
                <button className="mcc-port-action-btn" onClick={e => { e.stopPropagation(); onSelect(); }}>Open</button>
                <button className="mcc-port-action-btn mcc-port-stop" onClick={e => { e.stopPropagation(); onRemove(); }}>Remove</button>
            </div>
        </div>
    );
}

// HomeView with Outlet for nested routes
export function HomeView() {
    return <Outlet />;
}

// Icons
function PlusIcon() {
    return (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <line x1="12" y1="5" x2="12" y2="19" />
            <line x1="5" y1="12" x2="19" y2="12" />
        </svg>
    );
}

function GitIcon() {
    return (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <circle cx="18" cy="18" r="3" />
            <circle cx="6" cy="6" r="3" />
            <path d="M13 6h3a2 2 0 0 1 2 2v7" />
            <line x1="6" y1="9" x2="6" y2="21" />
        </svg>
    );
}

function DiagnoseIcon() {
    return (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M9 11l3 3L22 4" />
            <path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11" />
        </svg>
    );
}
