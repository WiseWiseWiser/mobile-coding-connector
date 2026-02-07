import { useState, useEffect } from 'react';
import { Outlet } from 'react-router-dom';
import { useCurrent } from '../../../hooks/useCurrent';
import { useTabNavigate } from '../../../hooks/useTabNavigate';
import { NavTabs } from '../types';
import {
    fetchAgents, fetchAgentSessions, launchAgentSession, stopAgentSession,
    AgentSessionStatuses,
} from '../../../api/agents';
import type { AgentDef, AgentSessionInfo } from '../../../api/agents';
import { useV2Context } from '../../V2Context';
import { AgentEmptyIcon } from '../../icons';
import './AgentView.css';

// ---- Outlet Context ----

export interface AgentOutletContext {
    projectName: string | null;
    agents: AgentDef[];
    agentsLoading: boolean;
    session: AgentSessionInfo | null;
    setSession: (session: AgentSessionInfo | null) => void;
    launchError: string;
    onLaunchHeadless: (agent: AgentDef) => void;
    onStopSession: () => void;
    navigateToView: (view: string) => void;
}

export function AgentLayout() {
    const { currentProject } = useV2Context();
    const projectDir = currentProject?.dir ?? null;
    const projectName = currentProject?.name ?? null;
    const navigateToView = useTabNavigate(NavTabs.Agent);

    const [agents, setAgents] = useState<AgentDef[]>([]);
    const [agentsLoading, setAgentsLoading] = useState(true);
    const [session, setSession] = useState<AgentSessionInfo | null>(null);
    const [launchError, setLaunchError] = useState('');

    // Fetch agents list
    useEffect(() => {
        fetchAgents()
            .then(data => { setAgents(data); setAgentsLoading(false); })
            .catch(() => setAgentsLoading(false));
    }, []);

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

    const handleLaunchHeadless = async (agent: AgentDef) => {
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
