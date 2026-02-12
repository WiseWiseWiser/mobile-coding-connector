import { useState } from 'react';
import { Outlet, useNavigate, useOutletContext } from 'react-router-dom';
import { deleteProject as apiDeleteProject } from '../../../api/projects';
import type { ProjectInfo } from '../../../api/projects';
import { cloneRepo } from '../../../api/auth';
import { useV2Context } from '../../V2Context';
import { useStreamingAction } from '../../../hooks/useStreamingAction';
import { validateProjectSSHKey } from '../../../hooks/useSSHKeyValidation';
import { StreamingLogs } from '../../StreamingComponents';
import { PlusIcon, GitIcon, DiagnoseIcon, SettingsIcon, FolderIcon, TerminalIcon } from '../../icons';
import { loadSSHKeys } from './settings/gitStorage';
import { encryptWithServerKey } from './crypto';
import { SSHKeyRequiredHint } from '../components/SSHKeyRequiredHint';

// Re-export sub-views for route registration
export { DiagnoseView } from './DiagnoseView';
export { SettingsView } from './SettingsView';
export { GitSettings } from './settings/GitSettings';
export { CloneRepoView } from './settings/CloneRepoView';
export { SSHServersView } from './SSHServersView';
export { ManageFilesView } from './ManageFilesView';

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

    // Clone streaming state - shared across all project cards
    const [cloningProjectId, setCloningProjectId] = useState<string | null>(null);
    const [cloneState, cloneControls] = useStreamingAction((result) => {
        if (result.ok) {
            fetchProjects();
        }
        setCloningProjectId(null);
    });

    const handleRemoveProject = async (id: string) => {
        try {
            await apiDeleteProject(id);
            fetchProjects();
        } catch {
            // ignore
        }
    };

    const handleClone = async (project: ProjectInfo) => {
        if (!project.repo_url) return;
        setCloningProjectId(project.id);

        cloneControls.run(async () => {
            const body: Record<string, unknown> = {
                repo_url: project.repo_url,
                target_dir: project.dir,
            };

            if (project.use_ssh && project.ssh_key_id) {
                const sshKeys = loadSSHKeys();
                const key = sshKeys.find(k => k.id === project.ssh_key_id);
                if (key) {
                    body.ssh_key = await encryptWithServerKey(key.privateKey);
                    body.use_ssh = true;
                    body.ssh_key_id = project.ssh_key_id;
                }
            }

            return cloneRepo(body);
        });
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
                            onOpen={() => {
                                navigate(`/project/${encodeURIComponent(project.name)}`);
                            }}
                            onRemove={() => handleRemoveProject(project.id)}
                            onClone={() => handleClone(project)}
                            cloning={cloningProjectId === project.id}
                            anyCloning={cloneState.running}
                        />
                    ))}
                    {/* Streaming logs area - shown below all cards */}
                    {(cloneState.showLogs || cloneState.result) && (
                        <div style={{ marginTop: 8 }}>
                            <StreamingLogs
                                state={cloneState}
                                pendingMessage="Cloning in progress..."
                                maxHeight={200}
                            />
                        </div>
                    )}
                </div>
            )}
            <button className="mcc-new-workspace-btn" onClick={() => navigate('clone-repo')}>
                <PlusIcon />
                <span>Clone Repository</span>
            </button>
            <button className="mcc-new-workspace-btn" onClick={() => navigate('add-from-filesystem')}>
                <FolderIcon />
                <span>Add From Filesystem</span>
            </button>
            <button className="mcc-git-settings-btn" onClick={() => navigate('settings/git')}>
                <GitIcon />
                <span>Git Settings</span>
            </button>
            <button className="mcc-diagnose-btn" onClick={() => navigate('diagnose')}>
                <DiagnoseIcon />
                <span>System Diagnostics</span>
            </button>
            <button className="mcc-diagnose-btn" onClick={() => navigate('manage-server')}>
                <SettingsIcon />
                <span>Manage Server</span>
            </button>
            <button className="mcc-diagnose-btn" onClick={() => navigate('ssh-servers')}>
                <TerminalIcon />
                <span>Manage SSH</span>
            </button>
        </div>
    );
}

// Project Card
interface ProjectCardProps {
    project: ProjectInfo;
    isActive: boolean;
    onSelect: () => void;
    onOpen: () => void;
    onRemove: () => void;
    onClone: () => void;
    cloning: boolean;
    anyCloning: boolean;
}

function ProjectCard({ project, isActive, onSelect, onOpen, onRemove, onClone, cloning, anyCloning }: ProjectCardProps) {
    const createdDate = new Date(project.created_at).toLocaleDateString();
    const dirMissing = !project.dir_exists;
    const sshValidation = validateProjectSSHKey(project);

    return (
        <div className={`mcc-workspace-card mcc-workspace-card-clickable${isActive ? ' mcc-workspace-card-active' : ''}`} onClick={onSelect}>
            <div className="mcc-workspace-card-header">
                <span className="mcc-workspace-name">{project.name}</span>
                {isActive && <span className="mcc-workspace-active-badge">Working on</span>}
                {dirMissing && <span className="mcc-workspace-missing-badge">Not cloned</span>}
            </div>
            <div className="mcc-workspace-card-meta">
                <span>{project.dir}</span>
            </div>
            <div className="mcc-workspace-card-meta" style={{ marginTop: 4 }}>
                <span>{project.repo_url}</span>
                <span>{createdDate}</span>
            </div>
            {dirMissing && project.repo_url && !sshValidation.hasSSHKey && (
                <div style={{ marginTop: 8 }}>
                    <SSHKeyRequiredHint message={sshValidation.message} style={{ marginBottom: 0 }} />
                </div>
            )}
            <div className="mcc-port-actions" style={{ marginTop: 8 }}>
                <button className="mcc-port-action-btn" onClick={e => { e.stopPropagation(); onOpen(); }}>Open</button>
                {dirMissing && project.repo_url && (
                    <button
                        className="mcc-port-action-btn mcc-port-clone"
                        onClick={e => { e.stopPropagation(); onClone(); }}
                        disabled={anyCloning || !sshValidation.hasSSHKey}
                        style={{ opacity: !sshValidation.hasSSHKey ? 0.5 : 1 }}
                    >
                        {cloning ? 'Cloning...' : 'Clone'}
                    </button>
                )}
                <button className="mcc-port-action-btn mcc-port-stop" onClick={e => { e.stopPropagation(); onRemove(); }}>Remove</button>
            </div>
        </div>
    );
}

// HomeView with Outlet for nested routes
export function HomeView() {
    return <Outlet />;
}

