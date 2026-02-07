import { useParams, useOutletContext } from 'react-router-dom';
import type { FilesOutletContext } from './FilesLayout';
import { FileContentView } from './FileContentView';

export function FileContentRoute() {
    const params = useParams<{ '*': string }>();
    const { projectDir, navigateToView } = useOutletContext<FilesOutletContext>();
    const filePath = params['*'] || '';

    return (
        <FileContentView
            projectDir={projectDir}
            filePath={filePath}
            onBack={() => {
                // Go back to the parent directory in browse view
                const parentDir = filePath.includes('/')
                    ? 'browse/' + filePath.substring(0, filePath.lastIndexOf('/'))
                    : 'browse';
                navigateToView(parentDir);
            }}
        />
    );
}
