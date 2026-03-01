import { useEffect, useCallback, useState } from 'react';
import { Outlet } from 'react-router-dom';
import { useCurrent } from '../../../hooks/useCurrent';
import { useTabNavigate } from '../../../hooks/useTabNavigate';
import { NavTabs } from '../types';
import {
    fetchAgentSessions, launchAgentSession, stopAgentSession,
    AgentSessionStatuses,
} from '../../../api/agents';
import type { AgentDef, AgentSessionInfo, ExternalOpencodeSession } from '../../../api/agents';
import { useV2Context } from '../../V2Context';
import { AgentEmptyIcon } from '../../../pure-view/icons/AgentEmptyIcon';
import { loadCursorAPIKey } from './cursorStorage';
import './AgentView.css';

// ---- Outlet Context ----

export interface AgentOutletContext {
    projectName: string | null;
    agents: AgentDef[];
    agentsLoading: boolean;
    sessions: Record<string, AgentSessionInfo>;
    setSession: (agentId: string, session: AgentSessionInfo | null) => void;
    launchError: string;
    sessionsLoadError: string | null;
    onLaunchHeadless: (agent: AgentDef) => void;
    onStopAgent: (agentId: string) => void;
    onRefreshAgents: () => void;
    navigateToView: (view: string) => void;
    // External sessions from CLI/web opencode
    externalSessions: ExternalOpencodeSession[];
    externalSessionsLoading: boolean;
    externalSessionsTotal: number;
    externalSessionsPage: number;
    refreshExternalSessions: (page: number) => void;
}

export function AgentLayout() {
    const {
        currentProject,
        agents,
        agentsLoading,
        refreshAgents,
        agentSessions: sessions,
        setAgentSession: setSession,
        agentLaunchError: launchError,
        setAgentLaunchError: setLaunchError,
        externalSessions,
        externalSessionsLoading,
        externalSessionsTotal,
        externalSessionsPage,
        refreshExternalSessions,
    } = useV2Context();
    const projectDir = currentProject?.dir ?? null;
    const projectName = currentProject?.name ?? null;
    const navigateToView = useTabNavigate(NavTabs.Agent);

    // Check for existing sessions matching this project
    const [sessionsLoadError, setSessionsLoadError] = useState<string | null>(null);
    const projectDirRef = useCurrent(projectDir);
    const setSessionRef = useCurrent(setSession);
    const loadSessions = useCallback(() => {
        if (!projectDir) return Promise.resolve();
        setSessionsLoadError(null);
        return fetchAgentSessions()
            .then(allSessions => {
                const active = allSessions.filter(
                    s => s.project_dir === projectDirRef.current &&
                        (s.status === AgentSessionStatuses.Running || s.status === AgentSessionStatuses.Starting)
                );
                for (const s of active) {
                    setSessionRef.current(s.agent_id, s);
                }
            })
            .catch((err) => {
                const errorMsg = err instanceof Error ? err.message : String(err);
                console.error('Failed to load agent sessions:', errorMsg);
                setSessionsLoadError(errorMsg);
            });
    }, [projectDir]);

    useEffect(() => {
        loadSessions();
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [projectDir]);

    // Also check for sessions when the tab becomes visible (e.g., when navigating back)
    useEffect(() => {
        const handleVisibilityChange = () => {
            if (document.visibilityState === 'visible') {
                loadSessions();
            }
        };
        document.addEventListener('visibilitychange', handleVisibilityChange);
        return () => document.removeEventListener('visibilitychange', handleVisibilityChange);
    }, [loadSessions]);

    const handleLaunchHeadless = async (agent: AgentDef) => {
        if (!projectDir) return;
        setLaunchError('');
        try {
            // For cursor-agent, pass the API key from localStorage
            const apiKey = agent.id === 'cursor-agent' ? loadCursorAPIKey() : undefined;
            const sessionInfo = await launchAgentSession(agent.id, projectDir, apiKey);
            setSession(agent.id, sessionInfo);
            navigateToView(agent.id);
        } catch (err) {
            setLaunchError(err instanceof Error ? err.message : String(err));
        }
    };

    const handleStopAgent = async (agentId: string) => {
        const session = sessions[agentId];
        if (!session) return;
        try {
            await stopAgentSession(session.id);
        } catch { /* ignore */ }
        setSession(agentId, null);
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
        sessions,
        setSession,
        launchError,
        sessionsLoadError,
        onLaunchHeadless: handleLaunchHeadless,
        onStopAgent: handleStopAgent,
        onRefreshAgents: refreshAgents,
        navigateToView,
        externalSessions,
        externalSessionsLoading,
        externalSessionsTotal,
        externalSessionsPage,
        refreshExternalSessions,
    };

    return (
        <>
            {sessionsLoadError && (
                <div className="mcc-agent-error" style={{ margin: '12px 16px', padding: '12px 16px', background: '#fef2f2', border: '1px solid #fecaca', borderRadius: '8px' }}>
                    <strong>Error loading sessions:</strong> {sessionsLoadError}
                </div>
            )}
            <Outlet context={ctx} />
        </>
    );
}
