import { useParams, useOutletContext } from 'react-router-dom';
import type { FilesOutletContext } from './FilesLayout';
import { FileBrowserView } from './FileBrowserView';

export function FileBrowserRoute() {
    const params = useParams<{ '*': string }>();
    const { projectDir, sshKeyId, navigateToView } = useOutletContext<FilesOutletContext>();
    const currentPath = params['*'] || '';

    return (
        <FileBrowserView
            projectDir={projectDir}
            currentPath={currentPath}
            sshKeyId={sshKeyId}
            onNavigate={(path) => navigateToView(path ? `browse/${path}` : 'browse')}
            onViewFile={(path) => navigateToView(`file/${path}`)}
        />
    );
}
