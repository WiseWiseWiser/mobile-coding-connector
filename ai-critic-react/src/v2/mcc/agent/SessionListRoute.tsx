import { useOutletContext, useParams } from 'react-router-dom';
import type { AgentOutletContext } from './AgentLayout';
import { SessionList } from './SessionList';
import { AgentPicker } from './AgentPicker';
import { ExternalSessionList } from './ExternalSessionList';

export function SessionListRoute() {
    const ctx = useOutletContext<AgentOutletContext>();
    const params = useParams<{ agentId?: string }>();
    const agentId = params.agentId || '';

    const session = ctx.sessions[agentId];

    // For opencode with external sessions but no internal session, show external session list
    const hasExternalSessions = agentId === 'opencode' && ctx.externalSessions.length > 0;
    
    if (!session && hasExternalSessions) {
        return (
            <ExternalSessionList
                projectName={ctx.projectName}
                onBack={() => ctx.navigateToView('')}
                onSelectSession={(sessionId) => {
                    // For external sessions, we need to open them in a new tab/window
                    // since we can't proxy through the agent session
                    const baseUrl = window.location.origin;
                    window.open(`${baseUrl}/agent/opencode/${sessionId}`, '_blank');
                }}
            />
        );
    }

    // If no session for this agent and no external sessions, fall back to agent picker
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
