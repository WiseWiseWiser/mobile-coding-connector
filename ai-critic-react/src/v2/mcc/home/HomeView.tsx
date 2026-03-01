import { useState } from 'react';
import { Outlet, useNavigate, useOutletContext } from 'react-router-dom';
import { deleteProject as apiDeleteProject } from '../../../api/projects';
import type { ProjectInfo } from '../../../api/projects';
import { cloneRepo } from '../../../api/auth';
import { useV2Context } from '../../V2Context';
import { useStreamingAction } from '../../../hooks/useStreamingAction';
import { validateProjectSSHKey } from '../../../hooks/useSSHKeyValidation';
import { StreamingLogs } from '../../StreamingComponents';
import { PlusIcon } from '../../../pure-view/icons/PlusIcon';
import { GitIcon } from '../../../pure-view/icons/GitIcon';
import { DiagnoseIcon } from '../../../pure-view/icons/DiagnoseIcon';
import { SettingsIcon } from '../../../pure-view/icons/SettingsIcon';
import { FolderIcon } from '../../../pure-view/icons/FolderIcon';
import { TerminalIcon } from '../../../pure-view/icons/TerminalIcon';
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
export { ExperimentalView } from './experimental/ExperimentalView';

interface HomeOutletContext {
    onSelectProject: (project: ProjectInfo) => void;
}

interface WorkspaceListViewProps {
    onSelectProject?: (project: ProjectInfo) => void;
}

export function WorkspaceListView({ onSelectProject: propOnSelectProject }: WorkspaceListViewProps) {
    const navigate = useNavigate();
    const { rootProjects, getSubProjectsCount, projectsLoading, fetchProjects, currentProject } = useV2Context();
    
    const outletContext = useOutletContext<HomeOutletContext | null>();
    const onSelectProject = propOnSelectProject ?? outletContext?.onSelectProject ?? (() => {});

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
            ) : rootProjects.length === 0 ? (
                <div className="mcc-ports-empty">No projects yet. Clone a repository from Git Settings.</div>
            ) : (
                <div className="mcc-workspace-cards">
                    {rootProjects.map(project => (
                        <ProjectCard
                            key={project.id}
                            project={project}
                            isActive={project.name === currentProject?.name}
                            subProjectsCount={getSubProjectsCount(project.id)}
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
            <button className="mcc-diagnose-btn" onClick={() => navigate('tools')}>
                <DiagnoseIcon />
                <span>Server Tools</span>
            </button>
            <button className="mcc-diagnose-btn" onClick={() => navigate('manage-server')}>
                <SettingsIcon />
                <span>Manage Server</span>
            </button>
            <button className="mcc-diagnose-btn" onClick={() => navigate('ssh-servers')}>
                <TerminalIcon />
                <span>Manage SSH</span>
            </button>
            <button className="mcc-experimental-btn" onClick={() => navigate('experimental')}>
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M10.42 12.06a2.999 2.999 0 0 1 3.16-3.16c.36-.03.72-.13 1.07-.28.35-.15.67-.36.94-.62.27-.27.48-.59.62-.94.15-.35.25-.71.28-1.07a2.999 2.999 0 0 1 3.16-3.16"/>
                    <path d="M14.5 12.06c-.03.36-.13.72-.28 1.07-.15.35-.36.67-.62.94-.27.27-.59.48-.94.62-.35.15-.71.25-1.07.28a2.999 2.999 0 0 1-3.16 3.16"/>
                    <path d="M12 19.06v2"/>
                    <path d="M12 15.06v2"/>
                    <path d="M8 17.06h2"/>
                    <path d="M14 17.06h2"/>
                </svg>
                <span>Experimental</span>
            </button>
        </div>
    );
}

// Project Card
interface ProjectCardProps {
    project: ProjectInfo;
    isActive: boolean;
    subProjectsCount: number;
    onSelect: () => void;
    onOpen: () => void;
    onRemove: () => void;
    onClone: () => void;
    cloning: boolean;
    anyCloning: boolean;
}

function ProjectCard({ project, isActive, subProjectsCount, onSelect, onOpen, onRemove, onClone, cloning, anyCloning }: ProjectCardProps) {
    const createdDate = new Date(project.created_at).toLocaleDateString();
    const dirMissing = !project.dir_exists;
    const sshValidation = validateProjectSSHKey(project);
    const gitStatus = project.git_status;
    const hasUncommitted = gitStatus && !gitStatus.is_clean && gitStatus.uncommitted > 0;

    return (
        <div className={`mcc-workspace-card mcc-workspace-card-clickable${isActive ? ' mcc-workspace-card-active' : ''}`} onClick={onSelect}>
            <div className="mcc-workspace-card-header">
                <span className="mcc-workspace-name">{project.name}</span>
                {subProjectsCount > 0 && <span className="mcc-workspace-subprojects-badge">{subProjectsCount} sub-project{subProjectsCount !== 1 ? 's' : ''}</span>}
                {isActive && <span className="mcc-workspace-active-badge">Working on</span>}
                {dirMissing && <span className="mcc-workspace-missing-badge">Not cloned</span>}
                {hasUncommitted && <span className="mcc-workspace-uncommitted-badge">{gitStatus.uncommitted} files uncommitted</span>}
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

