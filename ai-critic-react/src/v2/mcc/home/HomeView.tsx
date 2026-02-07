import { Outlet, useNavigate, useOutletContext } from 'react-router-dom';
import { deleteProject as apiDeleteProject } from '../../../api/projects';
import type { ProjectInfo } from '../../../api/projects';
import { useV2Context } from '../../V2Context';
import { PlusIcon, GitIcon, DiagnoseIcon, UploadIcon } from '../../icons';

// Re-export sub-views for route registration
export { DiagnoseView } from './DiagnoseView';
export { SettingsView } from './SettingsView';
export { GitSettings } from './settings/GitSettings';
export { CloneRepoView } from './settings/CloneRepoView';

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
            <button className="mcc-git-settings-btn" onClick={() => navigate('settings/git')}>
                <GitIcon />
                <span>Git Settings</span>
            </button>
            <button className="mcc-diagnose-btn" onClick={() => navigate('diagnose')}>
                <DiagnoseIcon />
                <span>System Diagnostics</span>
            </button>
            <button className="mcc-upload-btn" onClick={() => navigate('upload-file')}>
                <UploadIcon />
                <span>Upload File</span>
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

