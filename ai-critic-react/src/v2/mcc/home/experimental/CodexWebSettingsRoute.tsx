import { useNavigate } from 'react-router-dom';
import { useV2Context } from '../../../V2Context';
import { CodexSettingsView } from '../../agent/CodexSettingsView';

export function CodexWebSettingsRoute() {
    const navigate = useNavigate();
    const { currentProject } = useV2Context();
    return (
        <CodexSettingsView
            projectName={currentProject?.name ?? null}
            onBack={() => navigate('../codex-web')}
        />
    );
}
