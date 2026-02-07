import { useOutletContext } from 'react-router-dom';
import type { AgentOutletContext } from './AgentLayout';
import { AgentPicker } from './AgentPicker';

export function AgentPickerRoute() {
    const ctx = useOutletContext<AgentOutletContext>();

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
