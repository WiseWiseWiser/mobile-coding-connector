import { useEffect } from 'react';
import { Outlet } from 'react-router-dom';
import { useCurrent } from '../../../hooks/useCurrent';
import { useTabNavigate } from '../../../hooks/useTabNavigate';
import { NavTabs } from '../types';
import {
    fetchAgentSessions, launchAgentSession, stopAgentSession,
    AgentSessionStatuses,
} from '../../../api/agents';
import type { AgentDef } from '../../../api/agents';
import { useV2Context } from '../../V2Context';
import { AgentEmptyIcon } from '../../icons';
import './AgentView.css';

// ---- Outlet Context ----

export interface AgentOutletContext {
    projectName: string | null;
    agents: AgentDef[];
    agentsLoading: boolean;
    session: import('../../../api/agents').AgentSessionInfo | null;
    setSession: (session: import('../../../api/agents').AgentSessionInfo | null) => void;
    launchError: string;
    onLaunchHeadless: (agent: AgentDef) => void;
    onStopSession: () => void;
    navigateToView: (view: string) => void;
}

export function AgentLayout() {
    const {
        currentProject,
        agents,
        agentsLoading,
        agentSession: session,
        setAgentSession: setSession,
        agentLaunchError: launchError,
        setAgentLaunchError: setLaunchError,
    } = useV2Context();
    const projectDir = currentProject?.dir ?? null;
    const projectName = currentProject?.name ?? null;
    const navigateToView = useTabNavigate(NavTabs.Agent);

    // Check for existing sessions matching this project
    const projectDirRef = useCurrent(projectDir);
    useEffect(() => {
        if (!projectDir) return;
        fetchAgentSessions()
            .then(sessions => {
                const existing = sessions.find(
                    s => s.project_dir === projectDirRef.current &&
                        (s.status === AgentSessionStatuses.Running || s.status === AgentSessionStatuses.Starting)
                );
                if (existing) {
                    setSession(existing);
                }
            })
            .catch(() => {});
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [projectDir]);

    const handleLaunchHeadless = async (agent: import('../../../api/agents').AgentDef) => {
        if (!projectDir) return;
        setLaunchError('');
        try {
            const sessionInfo = await launchAgentSession(agent.id, projectDir);
            setSession(sessionInfo);
            navigateToView(agent.id);
        } catch (err) {
            setLaunchError(err instanceof Error ? err.message : String(err));
        }
    };

    const handleStopSession = async () => {
        if (!session) return;
        try {
            await stopAgentSession(session.id);
        } catch { /* ignore */ }
        setSession(null);
        navigateToView('');
    };

    // No project selected → show empty state
    if (!projectDir) {
        return (
            <div className="mcc-agent-view">
                <div className="mcc-empty-state">
                    <AgentEmptyIcon />
                    <h3>No Project Selected</h3>
                    <p>Select a project from the Home tab to start an agent.</p>
                </div>
            </div>
        );
    }

    const ctx: AgentOutletContext = {
        projectName,
        agents,
        agentsLoading,
        session,
        setSession,
        launchError,
        onLaunchHeadless: handleLaunchHeadless,
        onStopSession: handleStopSession,
        navigateToView,
    };

    return <Outlet context={ctx} />;
}
