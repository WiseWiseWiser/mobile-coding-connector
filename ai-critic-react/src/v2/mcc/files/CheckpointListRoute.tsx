import { useOutletContext } from 'react-router-dom';
import type { FilesOutletContext } from './FilesLayout';
import { CheckpointListView } from './CheckpointListView';

export function CheckpointListRoute() {
    const { projectName, projectDir, sshKeyId, navigateToView } = useOutletContext<FilesOutletContext>();

    return (
        <CheckpointListView
            projectName={projectName}
            projectDir={projectDir}
            sshKeyId={sshKeyId}
            onCreateCheckpoint={() => navigateToView('create-checkpoint')}
            onSelectCheckpoint={(id) => navigateToView(`checkpoint/${id}`)}
            onGitCommit={() => navigateToView('git-commit')}
        />
    );
}
