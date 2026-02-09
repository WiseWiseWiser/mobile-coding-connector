import { useOutletContext, useParams } from 'react-router-dom';
import type { AgentOutletContext } from './AgentLayout';
import { SessionList } from './SessionList';
import { AgentPicker } from './AgentPicker';

export function SessionListRoute() {
    const ctx = useOutletContext<AgentOutletContext>();
    const params = useParams<{ agentId?: string }>();
    const agentId = params.agentId || '';

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
            />
        );
    }

    return (
        <SessionList
            session={session}
            projectName={ctx.projectName}
            onBack={() => ctx.navigateToView('')}
            onStop={() => ctx.onStopAgent(agentId)}
            onSelectSession={(sid) => ctx.navigateToView(`${agentId}/${sid}`)}
            onSessionUpdate={(updated) => ctx.setSession(agentId, updated)}
            onSettings={() => ctx.navigateToView(`${agentId}/settings`)}
        />
    );
}
