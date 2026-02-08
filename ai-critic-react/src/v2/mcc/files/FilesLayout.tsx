import { Outlet } from 'react-router-dom';
import { useTabNavigate } from '../../../hooks/useTabNavigate';
import { NavTabs } from '../types';
import { useV2Context } from '../../V2Context';
import './FilesView.css';

export interface FilesOutletContext {
    projectName: string;
    projectDir: string;
    sshKeyId?: string;
    navigateToView: (view?: string) => void;
}

export function FilesLayout() {
    const { currentProject } = useV2Context();
    const navigateToView = useTabNavigate(NavTabs.Files);

    if (!currentProject) {
        return (
            <div className="mcc-files">
                <div className="mcc-section-header"><h2>Files</h2></div>
                <div style={{ padding: '32px 16px', textAlign: 'center', color: '#94a3b8' }}>
                    Select a project first
                </div>
            </div>
        );
    }

    const ctx: FilesOutletContext = {
        projectName: currentProject.name,
        projectDir: currentProject.dir,
        sshKeyId: currentProject.ssh_key_id,
        navigateToView,
    };

    return (
        <div className="mcc-files">
            <Outlet context={ctx} />
        </div>
    );
}
