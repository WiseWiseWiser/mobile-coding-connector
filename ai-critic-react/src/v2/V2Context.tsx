import { createContext, useContext, useState, useEffect, useCallback, useMemo } from 'react';
import { usePortForwards } from '../hooks/usePortForwards';
import type { UsePortForwardsReturn } from '../hooks/usePortForwards';
import { useLocalPorts } from '../hooks/useLocalPorts';
import type { UseLocalPortsReturn } from '../hooks/useLocalPorts';
import { fetchProjects as apiFetchProjects } from '../api/projects';
import type { ProjectInfo } from '../api/projects';
import { fetchDiagnostics as apiFetchDiagnostics } from '../api/ports';
import type { DiagnosticsData } from '../api/ports';
import { fetchAgents, fetchExternalSessions } from '../api/agents';
import type { AgentDef, AgentSessionInfo, ExternalOpencodeSession } from '../api/agents';
import type { NavTab } from './mcc/types';

interface V2ContextValue {
    projectsList: ProjectInfo[];
    rootProjects: ProjectInfo[];
    getSubProjectsCount: (projectId: string) => number;
    projectsLoading: boolean;
    fetchProjects: () => void;
    currentProject: ProjectInfo | null;
    setCurrentProject: (project: ProjectInfo | null) => void;
    portForwards: UsePortForwardsReturn;
    localPorts: UseLocalPortsReturn;
    diagnostics: DiagnosticsData | null;
    diagnosticsLoading: boolean;
    refreshDiagnostics: () => void;
    agents: AgentDef[];
    agentsLoading: boolean;
    refreshAgents: () => void;
    agentSessions: Record<string, AgentSessionInfo>;
    setAgentSession: (agentId: string, session: AgentSessionInfo | null) => void;
    agentLaunchError: string;
    setAgentLaunchError: (error: string) => void;
    externalSessions: ExternalOpencodeSession[];
    externalSessionsLoading: boolean;
    externalSessionsTotal: number;
    externalSessionsPage: number;
    refreshExternalSessions: (page?: number) => void;
    tabHistories: Record<NavTab, string[]>;
    pushTabHistory: (tab: NavTab, path: string) => void;
    popTabHistory: (tab: NavTab) => string | undefined;
    clearTabHistory: (tab: NavTab) => void;
}

const V2Ctx = createContext<V2ContextValue | null>(null);

export function useV2Context(): V2ContextValue {
    const ctx = useContext(V2Ctx);
    if (!ctx) throw new Error('useV2Context must be used within V2Provider');
    return ctx;
}

export function V2Provider({ children }: { children: React.ReactNode }) {
    const [projectsList, setProjectsList] = useState<ProjectInfo[]>([]);
    const [projectsLoading, setProjectsLoading] = useState(true);

    const fetchProjects = () => {
        apiFetchProjects({ all: true })
            .then(data => { setProjectsList(data); setProjectsLoading(false); })
            .catch(() => setProjectsLoading(false));
    };

    useEffect(() => {
        fetchProjects();
    }, []);

    const rootProjects = useMemo(() => 
        projectsList.filter(p => !p.parent_id),
        [projectsList]
    );

    const subProjectsCountMap = useMemo(() => {
        const map = new Map<string, number>();
        for (const p of projectsList) {
            if (p.parent_id) {
                map.set(p.parent_id, (map.get(p.parent_id) || 0) + 1);
            }
        }
        return map;
    }, [projectsList]);

    const getSubProjectsCount = useCallback((projectId: string): number => {
        return subProjectsCountMap.get(projectId) || 0;
    }, [subProjectsCountMap]);

    const [currentProject, setCurrentProject] = useState<ProjectInfo | null>(null);

    // Port forwarding
    const portForwards = usePortForwards();

    // Local listening ports (SSE stream, persists across tab switches)
    const localPortsState = useLocalPorts();

    // Cloudflare diagnostics (fetched once, persists across tab switches)
    const [diagnostics, setDiagnostics] = useState<DiagnosticsData | null>(null);
    const [diagnosticsLoading, setDiagnosticsLoading] = useState(true);

    const refreshDiagnostics = () => {
        setDiagnosticsLoading(true);
        apiFetchDiagnostics()
            .then(d => { setDiagnostics(d); setDiagnosticsLoading(false); })
            .catch(() => setDiagnosticsLoading(false));
    };

    useEffect(() => {
        refreshDiagnostics();
    }, []);

    // Agent state (lifted up to persist across tab switches)
    const [agents, setAgents] = useState<AgentDef[]>([]);
    const [agentsLoading, setAgentsLoading] = useState(true);
    const [agentSessions, setAgentSessions] = useState<Record<string, AgentSessionInfo>>({});
    const [agentLaunchError, setAgentLaunchError] = useState('');
    // External agent sessions (opencode CLI/web sessions)
    const [externalSessions, setExternalSessions] = useState<ExternalOpencodeSession[]>([]);
    const [externalSessionsLoading, setExternalSessionsLoading] = useState(false);
    const [externalSessionsTotal, setExternalSessionsTotal] = useState(0);
    const [externalSessionsPage, setExternalSessionsPage] = useState(1);

    const setAgentSession = (agentId: string, session: AgentSessionInfo | null) => {
        setAgentSessions(prev => {
            const next = { ...prev };
            if (session) {
                next[agentId] = session;
            } else {
                delete next[agentId];
            }
            return next;
        });
    };

    const refreshAgents = () => {
        setAgentsLoading(true);
        fetchAgents()
            .then(data => { setAgents(data); setAgentsLoading(false); })
            .catch(() => setAgentsLoading(false));
    };

    useEffect(() => {
        refreshAgents();
    }, []);

    // External sessions refresh
    const refreshExternalSessions = useCallback((page: number = 1) => {
        setExternalSessionsLoading(true);
        // Fetch 5 sessions per page
        fetchExternalSessions(page, 5)
            .then(data => {
                if (data && data.items) {
                    setExternalSessions(data.items);
                    setExternalSessionsTotal(data.total || 0);
                    setExternalSessionsPage(page);
                } else {
                    setExternalSessions([]);
                    setExternalSessionsTotal(0);
                }
                setExternalSessionsLoading(false);
            })
            .catch(() => {
                setExternalSessions([]);
                setExternalSessionsTotal(0);
                setExternalSessionsLoading(false);
            });
    }, []);

    // Fetch external sessions on mount
    useEffect(() => {
        refreshExternalSessions();
    }, [refreshExternalSessions]);

    // Per-tab navigation history
    const [tabHistories, setTabHistories] = useState<Record<NavTab, string[]>>({} as Record<NavTab, string[]>);

    const pushTabHistory = useCallback((tab: NavTab, path: string) => {
        setTabHistories(prev => ({
            ...prev,
            [tab]: [...(prev[tab] || []), path],
        }));
    }, []);

    const popTabHistory = useCallback((tab: NavTab): string | undefined => {
        // Get the current value synchronously before updating
        const history = tabHistories[tab] || [];
        if (history.length === 0) return undefined;
        
        const popped = history[history.length - 1];
        setTabHistories(prev => ({
            ...prev,
            [tab]: (prev[tab] || []).slice(0, -1),
        }));
        return popped;
    }, [tabHistories]);

    const clearTabHistory = useCallback((tab: NavTab) => {
        setTabHistories(prev => ({
            ...prev,
            [tab]: [],
        }));
    }, []);

    return (
        <V2Ctx.Provider value={{
            projectsList,
            rootProjects,
            getSubProjectsCount,
            projectsLoading,
            fetchProjects,
            currentProject,
            setCurrentProject,
            portForwards,
            localPorts: localPortsState,
            diagnostics,
            diagnosticsLoading,
            refreshDiagnostics,
            agents,
            agentsLoading,
            refreshAgents,
            agentSessions,
            setAgentSession,
            agentLaunchError,
            setAgentLaunchError,
            externalSessions,
            externalSessionsLoading,
            externalSessionsTotal,
            externalSessionsPage,
            refreshExternalSessions,
            tabHistories,
            pushTabHistory,
            popTabHistory,
            clearTabHistory,
        }}>
            {children}
        </V2Ctx.Provider>
    );
}
