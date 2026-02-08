import { createContext, useContext, useState, useEffect } from 'react';
import { usePortForwards } from '../hooks/usePortForwards';
import type { UsePortForwardsReturn } from '../hooks/usePortForwards';
import { useLocalPorts } from '../hooks/useLocalPorts';
import type { UseLocalPortsReturn } from '../hooks/useLocalPorts';
import { fetchProjects as apiFetchProjects } from '../api/projects';
import type { ProjectInfo } from '../api/projects';
import { fetchDiagnostics as apiFetchDiagnostics } from '../api/ports';
import type { DiagnosticsData } from '../api/ports';
import { fetchAgents } from '../api/agents';
import type { AgentDef, AgentSessionInfo } from '../api/agents';

interface TerminalTab {
    id: string;
    name: string;
    cwd?: string;
    initialCommand?: string;
    sessionId?: string;
}

interface V2ContextValue {
    // Projects
    projectsList: ProjectInfo[];
    projectsLoading: boolean;
    fetchProjects: () => void;
    // Current project
    currentProject: ProjectInfo | null;
    setCurrentProject: (project: ProjectInfo | null) => void;
    // Port forwarding
    portForwards: UsePortForwardsReturn;
    // Local listening ports (SSE stream, persists across tab switches)
    localPorts: UseLocalPortsReturn;
    // Cloudflare diagnostics (fetched once, survives tab switches)
    diagnostics: DiagnosticsData | null;
    diagnosticsLoading: boolean;
    refreshDiagnostics: () => void;
    // Agent session state (lifted up to persist across tab switches)
    agents: AgentDef[];
    agentsLoading: boolean;
    agentSessions: Record<string, AgentSessionInfo>;
    setAgentSession: (agentId: string, session: AgentSessionInfo | null) => void;
    agentLaunchError: string;
    setAgentLaunchError: (error: string) => void;
    // Terminal state (lifted up to persist across tab switches)
    terminalTabs: TerminalTab[];
    setTerminalTabs: (tabs: TerminalTab[]) => void;
    activeTerminalTabId: string;
    setActiveTerminalTabId: (id: string) => void;
    terminalSessionsLoaded: boolean;
    setTerminalSessionsLoaded: (loaded: boolean) => void;
}

const V2Ctx = createContext<V2ContextValue | null>(null);

export function useV2Context(): V2ContextValue {
    const ctx = useContext(V2Ctx);
    if (!ctx) throw new Error('useV2Context must be used within V2Provider');
    return ctx;
}

export function V2Provider({ children }: { children: React.ReactNode }) {
    // Projects
    const [projectsList, setProjectsList] = useState<ProjectInfo[]>([]);
    const [projectsLoading, setProjectsLoading] = useState(true);

    const fetchProjects = () => {
        apiFetchProjects()
            .then(data => { setProjectsList(data); setProjectsLoading(false); })
            .catch(() => setProjectsLoading(false));
    };

    useEffect(() => {
        fetchProjects();
    }, []);

    // Current project
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

    useEffect(() => {
        fetchAgents()
            .then(data => { setAgents(data); setAgentsLoading(false); })
            .catch(() => setAgentsLoading(false));
    }, []);

    // Terminal state (lifted up to persist across tab switches)
    const [terminalTabs, setTerminalTabs] = useState<TerminalTab[]>([]);
    const [activeTerminalTabId, setActiveTerminalTabId] = useState('');
    const [terminalSessionsLoaded, setTerminalSessionsLoaded] = useState(false);

    return (
        <V2Ctx.Provider value={{
            projectsList,
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
            agentSessions,
            setAgentSession,
            agentLaunchError,
            setAgentLaunchError,
            terminalTabs,
            setTerminalTabs,
            activeTerminalTabId,
            setActiveTerminalTabId,
            terminalSessionsLoaded,
            setTerminalSessionsLoaded,
        }}>
            {children}
        </V2Ctx.Provider>
    );
}
