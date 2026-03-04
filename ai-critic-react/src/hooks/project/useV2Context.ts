import { createContext, useContext } from 'react';
import type { UsePortForwardsReturn } from '../usePortForwards';
import type { UseLocalPortsReturn } from '../useLocalPorts';
import type { ProjectInfo } from '../../api/projects';
import type { DiagnosticsData } from '../../api/ports';
import type { AgentDef, AgentSessionInfo, ExternalOpencodeSession } from '../../api/agents';
import type { NavTab } from '../../v2/mcc/types';
import type { ServerConfig } from '../../api/config';

export interface V2ContextValue {
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
    serverConfig: ServerConfig | null;
    serverConfigLoading: boolean;
}

export const V2Ctx = createContext<V2ContextValue | null>(null);

// NOTE: For the resolved project directory (respecting worktrees), use useProjectDir() instead of currentProject.dir.
export function useV2Context(): V2ContextValue {
    const ctx = useContext(V2Ctx);
    if (!ctx) throw new Error('useV2Context must be used within V2Provider');
    return ctx;
}
