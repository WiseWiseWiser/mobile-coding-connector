import { useOutletContext, useParams } from 'react-router-dom';
import type { AgentOutletContext } from './AgentLayout';
import { AgentChat } from './AgentChat';
import { AgentPicker } from './AgentPicker';

export function AgentChatRoute() {
    const ctx = useOutletContext<AgentOutletContext>();
    const params = useParams<{ agentId?: string; sessionId?: string }>();
    const agentId = params.agentId || '';
    const sessionId = params.sessionId || '';

    const session = ctx.sessions[agentId];

    // If no session for this agent, fall back to agent picker
    if (!session) {
        return (
            <AgentPicker
                agents={ctx.agents}
                loading={ctx.agentsLoading}
                launchError={ctx.launchError}
                sessions={ctx.sessions}
                onLaunchHeadless={ctx.onLaunchHeadless}
                onOpenSessions={(aid) => ctx.navigateToView(aid)}
                onStopAgent={ctx.onStopAgent}
                onConfigureAgent={(aid) => ctx.navigateToView(`${aid}/settings`)}
                externalSessions={ctx.externalSessions}
            />
        );
    }

    return (
        <AgentChat
            session={session}
            projectName={ctx.projectName}
            opencodeSID={sessionId}
            onStop={() => ctx.onStopAgent(agentId)}
            onBack={() => ctx.navigateToView(agentId)}
            onSessionUpdate={(updated) => ctx.setSession(agentId, updated)}
        />
    );
}
