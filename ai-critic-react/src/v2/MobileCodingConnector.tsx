import { useEffect, useRef, useState } from 'react';
import { useParams, useNavigate, useLocation, Outlet } from 'react-router-dom';
import { useCurrent } from '../hooks/useCurrent';
import type { ProjectInfo } from '../api/projects';
import { useV2Context } from './V2Context';
import { NavTabs } from './mcc/types';
import type { NavTab } from './mcc/types';
import { MenuIcon, SettingsIcon, ProfileIcon, HomeIcon, AgentIcon, TerminalIcon, PortsIcon, FilesIcon } from './icons';
import { NavButton } from './buttons';
import { ProjectDropdown } from './mcc/ProjectDropdown';
import { TerminalManager } from './mcc/terminal/TerminalManager';
import type { TerminalManagerHandle } from './mcc/terminal/TerminalManager';
import { fetchTerminalSessions } from '../api/terminal';
import './MobileCodingConnector.css';

export function MobileCodingConnector() {
    const params = useParams<{ projectName?: string; agentId?: string; sessionId?: string; view?: string; '*'?: string }>();
    const navigate = useNavigate();
    const location = useLocation();

    // Derive active tab and view from URL pathname (child route params propagate to layout)
    const projectNameFromUrl = params.projectName || '';
    const pathPrefix = projectNameFromUrl
        ? `/project/${encodeURIComponent(projectNameFromUrl)}/`
        : '/';
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

    const terminalManagerRef = useRef<TerminalManagerHandle>(null);

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
        if (proj) {
            const projBase = `/project/${encodeURIComponent(proj.name)}`;
            if (tab === NavTabs.Home && !view) return `${projBase}/home`;
            if (view) return `${projBase}/${tab}/${view}`;
            return `${projBase}/${tab}`;
        }
        // No project selected
        if (tab === NavTabs.Home && !view) return '/';
        if (view) return `/${tab}/${view}`;
        return `/${tab}`;
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
        // Just update the URL to reflect the project selection, stay on current tab
        navigate(`/project/${encodeURIComponent(project.name)}/home`);
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

    const [menuOpen, setMenuOpen] = useState(false);

    const handleMenuNavigate = (path: string) => {
        setMenuOpen(false);
        navigate(path);
    };

    return (
        <div className="mcc">
            {/* Top Bar */}
            <div className="mcc-topbar">
                <button className="mcc-menu-btn" onClick={() => setMenuOpen(!menuOpen)}>
                    <MenuIcon />
                </button>
                <ProjectDropdown
                    projects={projectsList}
                    currentProject={currentProject}
                    onProjectSelect={handleSelectProject}
                />
                <button className="mcc-profile-btn">
                    <ProfileIcon />
                </button>
            </div>

            {/* Sidebar Drawer */}
            <div className={`mcc-drawer-overlay${menuOpen ? ' mcc-drawer-overlay--open' : ''}`} onClick={() => setMenuOpen(false)} />
            <div className={`mcc-drawer${menuOpen ? ' mcc-drawer--open' : ''}`}>
                <div className="mcc-drawer-header">
                    <span className="mcc-drawer-title">Menu</span>
                    <button className="mcc-drawer-close" onClick={() => setMenuOpen(false)}>
                        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                            <line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" />
                        </svg>
                    </button>
                </div>
                <nav className="mcc-drawer-nav">
                    <button className="mcc-drawer-item" onClick={() => handleMenuNavigate(buildPath(NavTabs.Home, 'settings'))}>
                        <SettingsIcon />
                        <span>Settings</span>
                    </button>
                </nav>
            </div>

            {/* Main Content */}
            <div className="mcc-content">
                <div className="mcc-content-inner">
                    {/* Terminal - rendered persistently but hidden when not active */}
                    <div 
                        className="mcc-terminal-wrapper" 
                        style={{ 
                            display: activeTab === NavTabs.Terminal ? 'flex' : 'none',
                            flex: 1,
                            height: '100%',
                            overflow: 'hidden'
                        }}
                    >
                        <TerminalManager 
                            ref={terminalManagerRef} 
                            isVisible={activeTab === NavTabs.Terminal}
                            loadSessions={fetchTerminalSessions}
                        />
                    </div>
                    {/* Other tab content */}
                    <div 
                        className="mcc-tab-content"
                        style={{ 
                            display: activeTab === NavTabs.Terminal ? 'none' : 'flex',
                            flexDirection: 'column',
                            flex: 1,
                            minHeight: 0,
                            overflow: 'auto'
                        }}
                    >
                        <Outlet context={{ onSelectProject: handleSelectProject }} />
                    </div>
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
