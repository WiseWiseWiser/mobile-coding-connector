import { useParams, useOutletContext } from 'react-router-dom';
import type { AgentOutletContext } from './AgentLayout';
import { AgentPicker } from './AgentPicker';
import { CursorAgentSettings } from './CursorAgentSettings';

export function CursorAgentSettingsRoute() {
    const ctx = useOutletContext<AgentOutletContext>();
    const { agentId } = useParams<{ agentId: string }>();

    const session = agentId ? ctx.sessions[agentId] : undefined;
    if (!session) {
        return (
            <AgentPicker
                agents={ctx.agents}
                loading={ctx.agentsLoading}
                projectName={ctx.projectName}
                sessions={ctx.sessions}
                launchError={ctx.launchError}
                onLaunchHeadless={ctx.onLaunchHeadless}
                onOpenSessions={(id) => ctx.navigateToView(id)}
                onStopAgent={ctx.onStopAgent}
            />
        );
    }

    return (
        <CursorAgentSettings
            session={session}
            projectName={ctx.projectName}
            onBack={() => ctx.navigateToView(agentId!)}
        />
    );
}
