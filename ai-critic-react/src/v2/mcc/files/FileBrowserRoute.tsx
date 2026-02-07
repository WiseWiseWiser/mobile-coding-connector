import { useParams, useOutletContext } from 'react-router-dom';
import type { FilesOutletContext } from './FilesLayout';
import { FileBrowserView } from './FileBrowserView';

export function FileBrowserRoute() {
    const params = useParams<{ '*': string }>();
    const { projectDir, navigateToView } = useOutletContext<FilesOutletContext>();
    const currentPath = params['*'] || '';

    return (
        <FileBrowserView
            projectDir={projectDir}
            currentPath={currentPath}
            onNavigate={(path) => navigateToView(path ? `browse/${path}` : 'browse')}
            onViewFile={(path) => navigateToView(`file/${path}`)}
        />
    );
}
