import { useOutletContext } from 'react-router-dom';
import type { FilesOutletContext } from './FilesLayout';
import { ActionsView } from './ActionsView';

export function ActionsRoute() {
    const ctx = useOutletContext<FilesOutletContext>();

    return (
        <ActionsView
            projectName={ctx.projectName}
            projectDir={ctx.projectDir}
        />
    );
}
