import { useEffect, useRef, useState, useCallback } from 'react';
import { useNavigate, useLocation, Outlet } from 'react-router-dom';
import { useCurrent } from '../hooks/useCurrent';
import type { ProjectInfo } from '../api/projects';
import { useV2Context } from './V2Context';
import { NavTabs } from './mcc/types';
import type { NavTab } from './mcc/types';
import { MenuIcon, SettingsIcon, HomeIcon, AgentIcon, TerminalIcon, PortsIcon, FilesIcon, LogsIcon, BeakerIcon, ProjectDetailIcon } from './icons';
import { NavButton } from './buttons';
import { ProjectDropdown } from './mcc/ProjectDropdown';
import { TerminalManager } from './mcc/terminal/TerminalManager';
import type { TerminalManagerHandle } from './mcc/terminal/TerminalManager';
import { fetchTerminalSessions } from '../api/terminal';
import { WorktreeProvider, useWorktreeContext } from './context/WorktreeContext';
import { useWorktreeRoute } from './hooks/useWorktreeRoute';
import { useWorktreeManager } from './hooks/useWorktreeManager';
import { WorktreeSelector } from './components/WorktreeSelector';
import './MobileCodingConnector.css';

// Inner component that uses worktree context
function MobileCodingConnectorInner() {
    const navigate = useNavigate();
    const location = useLocation();

    // Parse worktree info from URL
    const { 
        projectName, 
        worktreeId, 
        fullProjectName,
        navigateToWorktree
    } = useWorktreeRoute();

    // Derive active tab and view from URL pathname
    const pathPrefix = fullProjectName
        ? `/project/${encodeURIComponent(fullProjectName)}/`
        : '/';
    const pathRest = location.pathname.startsWith(pathPrefix)
        ? location.pathname.slice(pathPrefix.length)
        : '';
    const slashIdx = pathRest.indexOf('/');
    const activeTab = ((slashIdx < 0 ? pathRest : pathRest.slice(0, slashIdx)) as NavTab) || NavTabs.Home;
    const viewFromUrl = slashIdx < 0 ? '' : pathRest.slice(slashIdx + 1);

    // Shared state from V2Context
    const {
        projectsList, projectsLoading,
        currentProject, setCurrentProject,
    } = useV2Context();

    // Worktree context
    const {
        currentWorktree,
        worktrees,
        setCurrentWorktree,
        getWorktreeById,
    } = useWorktreeContext();

    // Worktree manager
    const {
        loadWorktrees,
        loading: worktreesLoading,
    } = useWorktreeManager();

    const terminalManagerRef = useRef<TerminalManagerHandle>(null);

    // Restore project from URL on mount
    useEffect(() => {
        if (!projectName || projectsLoading || currentProject) return;
        const project = projectsList.find(p => p.name === projectName);
        if (project) {
            setCurrentProject(project);
        }
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [projectName, projectsLoading, projectsList]);

    // Load worktrees when project changes
    useEffect(() => {
        if (currentProject) {
            loadWorktrees(currentProject).then(() => {
                // Set current worktree based on URL worktreeId
                if (worktreeId !== undefined) {
                    const targetWorktree = worktrees.find(w => w.id === worktreeId);
                    if (targetWorktree) {
                        setCurrentWorktree(targetWorktree);
                    }
                } else {
                    // Default to root worktree (id=0)
                    const rootWorktree = worktrees.find(w => w.id === 0);
                    if (rootWorktree) {
                        setCurrentWorktree(rootWorktree);
                    }
                }
            });
        }
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [currentProject?.id]);

    // Handle worktree navigation
    const handleSelectWorktree = useCallback((selectedWorktreeId: number) => {
        if (selectedWorktreeId === currentWorktree?.id) return;
        
        const targetWorktree = getWorktreeById(selectedWorktreeId);
        if (!targetWorktree) return;
        
        setCurrentWorktree(targetWorktree);
        navigateToWorktree(selectedWorktreeId);
    }, [currentWorktree, getWorktreeById, setCurrentWorktree, navigateToWorktree]);

    // Handle SSH connection from navigation state
    useEffect(() => {
        const state = location.state as { sshConnect?: boolean; command?: string; serverName?: string } | null;
        if (state?.sshConnect && state?.command && terminalManagerRef.current) {
            // Switch to terminal tab
            if (activeTab !== NavTabs.Terminal) {
                navigate(buildPath(NavTabs.Terminal));
            }
            // Create new terminal tab with SSH command
            setTimeout(() => {
                terminalManagerRef.current?.openTab(
                    state.serverName || 'SSH Connection',
                    undefined,
                    state.command
                );
            }, 100);
            // Clear the navigation state
            navigate(location.pathname, { replace: true, state: {} });
        }
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [location.state, activeTab]);

    // Helper: build a path for navigation
    const currentProjectRef = useCurrent(currentProject);
    const currentWorktreeRef = useCurrent(currentWorktree);
    const buildPath = (tab: NavTab, view?: string): string => {
        const proj = currentProjectRef.current;
        const wt = currentWorktreeRef.current;
        if (proj) {
            // Build project name with worktree suffix if not root
            let projBase = `/project/${encodeURIComponent(proj.name)}`;
            if (wt && wt.id !== 0) {
                projBase += `~${wt.id}`;
            }
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
        // Stay on the current tab when switching projects
        const savedView = tabViewHistoryRef.current[activeTab];
        const projBase = `/project/${encodeURIComponent(project.name)}`;
        if (activeTab === NavTabs.Home && !savedView) {
            navigate(`${projBase}/home`);
        } else if (savedView) {
            navigate(`${projBase}/${activeTab}/${savedView}`);
        } else {
            navigate(`${projBase}/${activeTab}`);
        }
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

    // Build outlet context with worktree info
    const outletContext = {
        onSelectProject: handleSelectProject,
        projectName: currentProject?.name || null,
        projectDir: currentWorktree?.path || currentProject?.dir || null,
        worktreeId: currentWorktree?.id || null,
        worktreeBranch: currentWorktree?.branch || null,
        isWorktree: !!currentWorktree && !currentWorktree.isMain,
    };

    return (
        <div className="mcc">
            {/* Top Bar */}
            <div className="mcc-topbar">
                <button className="mcc-menu-btn" onClick={() => setMenuOpen(!menuOpen)}>
                    <MenuIcon />
                </button>
                <div className="mcc-project-section">
                    <ProjectDropdown
                        projects={projectsList}
                        currentProject={currentProject}
                        onProjectSelect={handleSelectProject}
                    />
                    {worktrees.length > 1 && (
                        <WorktreeSelector
                            worktrees={worktrees}
                            currentWorktree={currentWorktree}
                            onSelectWorktree={handleSelectWorktree}
                            disabled={worktreesLoading}
                        />
                    )}
                </div>
                <button className="mcc-profile-btn" onClick={() => currentProject && navigate(`/project/${encodeURIComponent(currentProject.name)}`)}>
                    <ProjectDetailIcon />
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
                    <button className="mcc-drawer-item" onClick={() => handleMenuNavigate(buildPath(NavTabs.Home, 'settings/logs'))}>
                        <LogsIcon />
                        <span>Logs</span>
                    </button>
                    <button className="mcc-drawer-item" onClick={() => handleMenuNavigate(buildPath(NavTabs.Home, 'experimental'))}>
                        <BeakerIcon />
                        <span>Experimental</span>
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
                            defaultCwd={currentWorktree?.path || currentProject?.dir}
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
                        <Outlet context={outletContext} />
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

// Wrapper component that provides WorktreeContext
export function MobileCodingConnector() {
    return (
        <WorktreeProvider>
            <MobileCodingConnectorInner />
        </WorktreeProvider>
    );
}
