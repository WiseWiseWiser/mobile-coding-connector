import { useParams, useOutletContext } from 'react-router-dom';
import type { AgentOutletContext } from './AgentLayout';
import { CursorAgentSettings } from './CursorAgentSettings';
import { AgentPathSettings } from './AgentPathSettings';
import { OpencodeSettings } from './OpencodeSettings';

export function AgentSettingsRoute() {
    const ctx = useOutletContext<AgentOutletContext>();
    const { agentId } = useParams<{ agentId: string }>();

    if (!agentId) {
        // No agent ID - go back to picker
        ctx.navigateToView('');
        return null;
    }

    const session = ctx.sessions[agentId];

    // If there's a running session, show the session-specific settings
    if (session) {
        // Use OpenCode-specific settings for opencode agent
        if (session.agent_id === 'opencode') {
            return (
                <OpencodeSettings
                    session={session}
                    projectName={ctx.projectName}
                    onBack={() => ctx.navigateToView(agentId)}
                />
            );
        }
        // Use Cursor-specific settings for cursor-agent (and other agents)
        return (
            <CursorAgentSettings
                session={session}
                projectName={ctx.projectName}
                onBack={() => ctx.navigateToView(agentId)}
            />
        );
    }

    // Otherwise, show the agent path configuration
    return (
        <AgentPathSettings
            agentId={agentId}
            onBack={() => ctx.navigateToView('')}
            onRefreshAgents={ctx.onRefreshAgents}
        />
    );
}
