import { useOutletContext, useParams } from 'react-router-dom';
import type { AgentOutletContext } from './AgentLayout';
import { AgentChat } from './AgentChat';
import { AgentPicker } from './AgentPicker';

export function AgentChatRoute() {
    const ctx = useOutletContext<AgentOutletContext>();
    const params = useParams<{ agentId?: string; sessionId?: string }>();
    const agentId = params.agentId || '';
    const sessionId = params.sessionId || '';

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
        <AgentChat
            session={ctx.session}
            projectName={ctx.projectName}
            opencodeSID={sessionId}
            onStop={ctx.onStopSession}
            onBack={() => ctx.navigateToView(agentId)}
            onSessionUpdate={ctx.setSession}
        />
    );
}
