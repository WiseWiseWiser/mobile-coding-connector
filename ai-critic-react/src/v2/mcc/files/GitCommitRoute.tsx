import { useOutletContext } from 'react-router-dom';
import type { FilesOutletContext } from './FilesLayout';
import { GitCommitView } from './GitCommitView';

export function GitCommitRoute() {
    const { projectName, projectDir, sshKeyId, navigateToView } = useOutletContext<FilesOutletContext>();

    return (
        <GitCommitView
            projectName={projectName}
            projectDir={projectDir}
            sshKeyId={sshKeyId}
            onBack={() => navigateToView('browse')}
        />
    );
}
