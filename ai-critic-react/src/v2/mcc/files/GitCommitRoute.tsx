import { useOutletContext } from 'react-router-dom';
import type { FilesOutletContext } from './FilesLayout';
import { GitCommitView } from './GitCommitView';

export function GitCommitRoute() {
    const { projectDir, navigateToView } = useOutletContext<FilesOutletContext>();

    return (
        <GitCommitView
            projectDir={projectDir}
            onBack={() => navigateToView('browse')}
        />
    );
}
