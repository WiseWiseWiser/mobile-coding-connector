import { useOutletContext, useParams } from 'react-router-dom';
import type { AgentOutletContext } from './AgentLayout';
import { SessionList } from './SessionList';
import { AgentPicker } from './AgentPicker';

export function SessionListRoute() {
    const ctx = useOutletContext<AgentOutletContext>();
    const params = useParams<{ agentId?: string }>();
    const agentId = params.agentId || '';

    // If no active session, fall back to agent picker
    if (!ctx.session) {
        return (
            <AgentPicker
                agents={ctx.agents}
                loading={ctx.agentsLoading}
                projectName={ctx.projectName}
                launchError={ctx.launchError}
                activeSession={ctx.session}
                onLaunchHeadless={ctx.onLaunchHeadless}
                onResumeChat={() => ctx.navigateToView(ctx.session?.agent_id || '')}
                onStopSession={ctx.onStopSession}
            />
        );
    }

    return (
        <SessionList
            session={ctx.session}
            projectName={ctx.projectName}
            onBack={() => ctx.navigateToView('')}
            onStop={ctx.onStopSession}
            onSelectSession={(sid) => ctx.navigateToView(`${agentId}/${sid}`)}
            onSessionUpdate={ctx.setSession}
        />
    );
}
