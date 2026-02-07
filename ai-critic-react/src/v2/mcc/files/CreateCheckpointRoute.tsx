import { useOutletContext } from 'react-router-dom';
import type { FilesOutletContext } from './FilesLayout';
import { CreateCheckpointView } from './CreateCheckpointView';

export function CreateCheckpointRoute() {
    const { projectName, projectDir, navigateToView } = useOutletContext<FilesOutletContext>();

    return (
        <CreateCheckpointView
            projectName={projectName}
            projectDir={projectDir}
            onBack={() => navigateToView()}
            onCreated={() => navigateToView()}
        />
    );
}
