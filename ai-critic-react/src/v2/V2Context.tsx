import { createContext, useContext, useState, useEffect } from 'react';
import { usePortForwards } from '../hooks/usePortForwards';
import type { UsePortForwardsReturn } from '../hooks/usePortForwards';
import { fetchProjects as apiFetchProjects } from '../api/projects';
import type { ProjectInfo } from '../api/projects';
import { fetchDiagnostics as apiFetchDiagnostics } from '../api/ports';
import type { DiagnosticsData } from '../api/ports';

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
    // Cloudflare diagnostics (fetched once, survives tab switches)
    diagnostics: DiagnosticsData | null;
    diagnosticsLoading: boolean;
    refreshDiagnostics: () => void;
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

    return (
        <V2Ctx.Provider value={{
            projectsList,
            projectsLoading,
            fetchProjects,
            currentProject,
            setCurrentProject,
            portForwards,
            diagnostics,
            diagnosticsLoading,
            refreshDiagnostics,
        }}>
            {children}
        </V2Ctx.Provider>
    );
}
