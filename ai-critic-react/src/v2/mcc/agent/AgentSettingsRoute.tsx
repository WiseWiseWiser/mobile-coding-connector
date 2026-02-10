import { useParams, useOutletContext } from 'react-router-dom';
import type { AgentOutletContext } from './AgentLayout';
import { CursorAgentSettings } from './CursorAgentSettings';
import { OpencodeSettings } from './OpencodeSettings';

export function AgentSettingsRoute() {
    const ctx = useOutletContext<AgentOutletContext>();
    const { agentId } = useParams<{ agentId: string }>();

    if (!agentId) {
        // No agent ID - go back to picker
        ctx.navigateToView('');
        return null;
    }

    const session = ctx.sessions[agentId] || null;
    const onBack = session
        ? () => ctx.navigateToView(agentId)
        : () => ctx.navigateToView('');

    // OpenCode agent always uses OpencodeSettings
    if (agentId === 'opencode') {
        return (
            <OpencodeSettings
                agentId={agentId}
                session={session}
                projectName={ctx.projectName}
                onBack={onBack}
                onRefreshAgents={ctx.onRefreshAgents}
            />
        );
    }

    // All other agents (cursor-agent, etc.) use CursorAgentSettings
    return (
        <CursorAgentSettings
            agentId={agentId}
            session={session}
            projectName={ctx.projectName}
            onBack={onBack}
            onRefreshAgents={ctx.onRefreshAgents}
        />
    );
}
