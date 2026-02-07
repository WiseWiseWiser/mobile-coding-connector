import { useParams, useOutletContext } from 'react-router-dom';
import type { FilesOutletContext } from './FilesLayout';
import { CheckpointDetailView } from './CheckpointDetailView';

export function CheckpointDetailRoute() {
    const { checkpointId } = useParams<{ checkpointId: string }>();
    const { projectName, navigateToView } = useOutletContext<FilesOutletContext>();
    const id = parseInt(checkpointId || '', 10);

    if (isNaN(id)) {
        return <div className="mcc-files-empty">Invalid checkpoint ID</div>;
    }

    return (
        <CheckpointDetailView
            projectName={projectName}
            checkpointId={id}
            onBack={() => navigateToView()}
        />
    );
}
