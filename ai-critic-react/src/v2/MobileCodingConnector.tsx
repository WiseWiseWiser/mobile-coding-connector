import { useEffect, useRef } from 'react';
import { useParams, useNavigate, useLocation, Outlet } from 'react-router-dom';
import { useCurrent } from '../hooks/useCurrent';
import type { ProjectInfo } from '../api/projects';
import { useV2Context } from './V2Context';
import { NavTabs } from './mcc/types';
import type { NavTab } from './mcc/types';
import { MenuIcon, SettingsIcon, ProfileIcon, HomeIcon, AgentIcon, TerminalIcon, PortsIcon, FilesIcon } from './icons';
import { NavButton } from './buttons';
import './MobileCodingConnector.css';

export function MobileCodingConnector() {
    const params = useParams<{ projectName?: string; agentId?: string; sessionId?: string; view?: string; '*'?: string }>();
    const navigate = useNavigate();
    const location = useLocation();

    // Derive active tab and view from URL pathname (child route params propagate to layout)
    const projectNameFromUrl = params.projectName || '';
    const pathPrefix = projectNameFromUrl
        ? `/v2/project/${encodeURIComponent(projectNameFromUrl)}/`
        : '/v2/';
    const pathRest = location.pathname.startsWith(pathPrefix)
        ? location.pathname.slice(pathPrefix.length)
        : '';
    const slashIdx = pathRest.indexOf('/');
    const activeTab = ((slashIdx < 0 ? pathRest : pathRest.slice(0, slashIdx)) as NavTab) || NavTabs.Home;
    const viewFromUrl = slashIdx < 0 ? '' : pathRest.slice(slashIdx + 1);

    // Shared state from V2Context (survives remounts across route changes)
    const {
        projectsList, projectsLoading,
        currentProject, setCurrentProject,
    } = useV2Context();

    // Restore project from URL on mount
    useEffect(() => {
        if (!projectNameFromUrl || projectsLoading || currentProject) return;
        const project = projectsList.find(p => p.name === projectNameFromUrl);
        if (project) {
            setCurrentProject(project);
        }
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [projectNameFromUrl, projectsLoading, projectsList]);

    // Helper: build a path for navigation
    const currentProjectRef = useCurrent(currentProject);
    const buildPath = (tab: NavTab, view?: string): string => {
        const proj = currentProjectRef.current;
        const base = '/v2';
        if (proj) {
            const projBase = `${base}/project/${encodeURIComponent(proj.name)}`;
            // Home tab with a project and no view shows the project list (no /tab suffix)
            if (tab === NavTabs.Home && !view) return projBase;
            if (view) return `${projBase}/${tab}/${view}`;
            return `${projBase}/${tab}`;
        }
        // No project selected
        if (tab === NavTabs.Home && !view) return base;
        if (view) return `${base}/${tab}/${view}`;
        return `${base}/${tab}`;
    };

    // Preserve route history per tab
    const tabViewHistoryRef = useRef<Record<string, string>>({});
    const activeTabRef = useCurrent(activeTab);
    const viewFromUrlRef = useCurrent(viewFromUrl);

    // Keep tab history in sync with current URL view
    useEffect(() => {
        if (viewFromUrl) {
            tabViewHistoryRef.current[activeTab] = viewFromUrl;
        }
    }, [activeTab, viewFromUrl]);

    const handleSelectProject = (project: ProjectInfo) => {
        setCurrentProject(project);
        navigate(`/v2/project/${encodeURIComponent(project.name)}/${NavTabs.Agent}`);
    };

    const handleTabChange = (tab: NavTab) => {
        // Save current view for the current tab before leaving
        const currentView = viewFromUrlRef.current;
        if (currentView) {
            tabViewHistoryRef.current[activeTabRef.current] = currentView;
        }
        // Restore saved view for the target tab
        const savedView = tabViewHistoryRef.current[tab];
        navigate(buildPath(tab, savedView || undefined));
    };

    return (
        <div className="mcc">
            {/* Top Bar */}
            <div className="mcc-topbar">
                <button className="mcc-menu-btn">
                    <MenuIcon />
                </button>
                <div className="mcc-title">
                    {currentProject ? currentProject.name : 'Mobile Coding Connector'}
                </div>
                <button className="mcc-settings-btn">
                    <SettingsIcon />
                </button>
                <button className="mcc-profile-btn">
                    <ProfileIcon />
                </button>
            </div>

            {/* Main Content */}
            <div className="mcc-content">
                <div className="mcc-content-inner">
                    <Outlet context={{ onSelectProject: handleSelectProject }} />
                </div>
            </div>

            {/* Bottom Navigation */}
            <div className="mcc-bottomnav">
                <NavButton
                    icon={<HomeIcon />}
                    label="Home"
                    active={activeTab === NavTabs.Home}
                    onClick={() => handleTabChange(NavTabs.Home)}
                />
                <NavButton
                    icon={<AgentIcon />}
                    label="Agent"
                    active={activeTab === NavTabs.Agent}
                    onClick={() => handleTabChange(NavTabs.Agent)}
                />
                <NavButton
                    icon={<TerminalIcon />}
                    label="Terminal"
                    active={activeTab === NavTabs.Terminal}
                    onClick={() => handleTabChange(NavTabs.Terminal)}
                />
                <NavButton
                    icon={<PortsIcon />}
                    label="Ports"
                    active={activeTab === NavTabs.Ports}
                    onClick={() => handleTabChange(NavTabs.Ports)}
                />
                <NavButton
                    icon={<FilesIcon />}
                    label="Files"
                    active={activeTab === NavTabs.Files}
                    onClick={() => handleTabChange(NavTabs.Files)}
                />
            </div>
        </div>
    );
}
