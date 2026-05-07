import { useOutletContext } from 'react-router-dom';
import type { AgentOutletContext } from './AgentLayout';
import { CodexSettingsView } from './CodexSettingsView';

export function AgentCodexSettingsRoute() {
    const ctx = useOutletContext<AgentOutletContext>();
    return (
        <CodexSettingsView
            projectName={ctx.projectName}
            onBack={() => ctx.navigateToView('codex-web')}
        />
    );
}
