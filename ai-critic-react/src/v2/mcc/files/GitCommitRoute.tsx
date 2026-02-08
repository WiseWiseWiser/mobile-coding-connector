import { useOutletContext } from 'react-router-dom';
import type { FilesOutletContext } from './FilesLayout';
import { GitCommitView } from './GitCommitView';

export function GitCommitRoute() {
    const { projectDir, sshKeyId, navigateToView } = useOutletContext<FilesOutletContext>();

    return (
        <GitCommitView
            projectDir={projectDir}
            sshKeyId={sshKeyId}
            onBack={() => navigateToView('browse')}
        />
    );
}
