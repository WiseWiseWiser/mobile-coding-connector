import { useOutletContext } from 'react-router-dom';
import type { AgentOutletContext } from './AgentLayout';
import { AgentPicker } from './AgentPicker';

export function AgentPickerRoute() {
    const ctx = useOutletContext<AgentOutletContext>();

    return (
        <AgentPicker
            agents={ctx.agents}
            loading={ctx.agentsLoading}
            launchError={ctx.launchError}
            sessions={ctx.sessions}
            onLaunchHeadless={ctx.onLaunchHeadless}
            onOpenSessions={(agentId) => ctx.navigateToView(agentId)}
            onStopAgent={ctx.onStopAgent}
            onConfigureAgent={(agentId) => ctx.navigateToView(`${agentId}/settings`)}
            externalSessions={ctx.externalSessions}
            onSelectSession={(sessionId) => ctx.navigateToView(`session/${sessionId}`)}
            onNewSession={() => ctx.navigateToView('new-session')}
        />
    );
}
