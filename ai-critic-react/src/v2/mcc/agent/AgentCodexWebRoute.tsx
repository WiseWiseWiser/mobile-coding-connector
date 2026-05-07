import { useOutletContext } from 'react-router-dom';
import type { AgentOutletContext } from './AgentLayout';
import { CodexCliChat } from './CodexCliChat';

export function AgentCodexWebRoute() {
    const ctx = useOutletContext<AgentOutletContext>();
    if (!ctx.projectDir) {
        return (
            <div className="mcc-agent-view">
                <div className="mcc-agent-error">No project directory selected.</div>
            </div>
        );
    }
    return (
        <CodexCliChat
            projectName={ctx.projectName}
            projectDir={ctx.projectDir}
            onBack={() => ctx.navigateToView('')}
            onSettings={() => ctx.navigateToView('codex-web/settings')}
        />
    );
}
