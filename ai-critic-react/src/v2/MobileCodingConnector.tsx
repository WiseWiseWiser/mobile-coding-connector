import { useState, useEffect, useRef } from 'react';
import { useParams, useNavigate, Outlet } from 'react-router-dom';
import { useCurrent } from '../hooks/useCurrent';
import type { PortForward, TunnelProvider, ProviderInfo } from '../hooks/usePortForwards';
import { PortStatuses, TunnelProviders } from '../hooks/usePortForwards';
import type { ProjectInfo } from '../api/projects';
import { fetchDiagnostics as apiFetchDiagnostics, fetchPortLogs as apiFetchPortLogs } from '../api/ports';
import type { DiagnosticsData } from '../api/ports';
import { useV2Context } from './V2Context';
import { TerminalManager } from './TerminalManager';
import type { TerminalManagerHandle } from './TerminalManager';
import { AgentView } from './AgentView';
import { FilesView as FilesViewComponent } from './FilesView';
import { LogViewer } from './LogViewer';
import './MobileCodingConnector.css';

// Navigation tabs
const NavTabs = {
    Home: 'home',
    Agent: 'agent',
    Terminal: 'terminal',
    Ports: 'ports',
    Files: 'files',
} as const;

type NavTab = typeof NavTabs[keyof typeof NavTabs];

export function MobileCodingConnector() {
    const params = useParams<{ tab?: string; view?: string; projectName?: string }>();
    const navigate = useNavigate();

    // Derive state from URL path params
    const activeTab = (params.tab as NavTab) || NavTabs.Home;
    // Combine view with wildcard rest for deep paths like files/browse/some/dir
    const viewBase = params.view || '';
    const viewRest = params['*'] || '';
    const viewFromUrl = viewRest ? `${viewBase}/${viewRest}` : viewBase;
    const projectNameFromUrl = params.projectName || '';

    // Shared state from V2Context (survives remounts across route changes)
    const {
        projectsList, projectsLoading,
        currentProject, setCurrentProject,
        portForwards: { ports, providers: availableProviders, loading: portsLoading, error: portsError, addPort, removePort, refresh: refreshPorts },
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

    // Port form state (local - ok to reset on remount)
    const [showNewPortForm, setShowNewPortForm] = useState(false);
    const [newPortNumber, setNewPortNumber] = useState('');
    const [newPortLabel, setNewPortLabel] = useState('');
    const [newPortProvider, setNewPortProvider] = useState<TunnelProvider>(TunnelProviders.Localtunnel);
    const [portActionError, setPortActionError] = useState<string | null>(null);

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

    const navigateToView = (view: string) => {
        const tab = activeTabRef.current;
        // Save this view in the tab history
        tabViewHistoryRef.current[tab] = view;
        navigate(buildPath(tab, view || undefined), { replace: true });
    };

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

    const handleAddPortForward = async () => {
        const portNum = parseInt(newPortNumber, 10);
        if (!portNum || portNum <= 0 || portNum > 65535) return;

        const label = newPortLabel || `Port ${portNum}`;
        const provider = newPortProvider;

        try {
            setPortActionError(null);
            await addPort(portNum, label, provider);
            // Dismiss form and refresh list only after the API call succeeds
            setShowNewPortForm(false);
            setNewPortNumber('');
            setNewPortLabel('');
            refreshPorts();
        } catch (err) {
            setPortActionError(err instanceof Error ? err.message : String(err));
        }
    };

    const handleRemovePort = async (port: number) => {
        try {
            setPortActionError(null);
            await removePort(port);
        } catch (err) {
            setPortActionError(err instanceof Error ? err.message : String(err));
        }
    };

    const renderContent = () => {
        switch (activeTab) {
            case NavTabs.Home:
                // Home tab uses Outlet for nested routes
                return <Outlet context={{ onSelectProject: handleSelectProject }} />;
            case NavTabs.Agent:
                return (
                    <AgentView
                        projectDir={currentProject?.dir ?? null}
                        projectName={currentProject?.name ?? null}
                        currentView={viewFromUrl}
                        onNavigateToView={navigateToView}
                    />
                );
            case NavTabs.Terminal:
                return null; // Terminal is rendered separately to persist state
            case NavTabs.Ports:
                return (
                    <PortForwardingView
                        ports={ports}
                        availableProviders={availableProviders}
                        loading={portsLoading}
                        error={portsError}
                        actionError={portActionError}
                        showNewForm={showNewPortForm}
                        onToggleNewForm={() => setShowNewPortForm(!showNewPortForm)}
                        newPortNumber={newPortNumber}
                        newPortLabel={newPortLabel}
                        newPortProvider={newPortProvider}
                        onPortNumberChange={setNewPortNumber}
                        onPortLabelChange={setNewPortLabel}
                        onPortProviderChange={setNewPortProvider}
                        onAddPort={handleAddPortForward}
                        onRemovePort={handleRemovePort}
                        currentView={viewFromUrl}
                        onNavigateToView={navigateToView}
                    />
                );
            case NavTabs.Files:
                if (!currentProject) {
                    return <div className="mcc-files"><div className="mcc-section-header"><h2>Files</h2></div><div style={{ padding: '32px 16px', textAlign: 'center', color: '#94a3b8' }}>Select a project first</div></div>;
                }
                return <FilesViewComponent projectName={currentProject.name} projectDir={currentProject.dir} view={viewFromUrl} onNavigateToView={navigateToView} />;
            default:
                return null;
        }
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
                {/* Regular content for non-terminal tabs */}
                <div className={`mcc-content-inner ${activeTab === NavTabs.Terminal ? 'hidden' : ''}`}>
                    {renderContent()}
                </div>
                {/* Terminal is ALWAYS mounted but hidden when not active - this preserves state */}
                <div className={`mcc-terminal-container ${activeTab === NavTabs.Terminal ? 'visible' : ''}`}>
                    <TerminalManager ref={terminalManagerRef} isVisible={true} />
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

// Port Forwarding View
interface PortForwardingViewProps {
    ports: PortForward[];
    availableProviders: ProviderInfo[];
    loading: boolean;
    error: string | null;
    actionError: string | null;
    showNewForm: boolean;
    onToggleNewForm: () => void;
    newPortNumber: string;
    newPortLabel: string;
    newPortProvider: TunnelProvider;
    onPortNumberChange: (value: string) => void;
    onPortLabelChange: (value: string) => void;
    onPortProviderChange: (value: TunnelProvider) => void;
    onAddPort: () => void;
    onRemovePort: (port: number) => void;
    currentView: string;
    onNavigateToView: (view: string) => void;
}

function PortForwardingView({
    ports,
    availableProviders,
    loading,
    error,
    actionError,
    showNewForm,
    onToggleNewForm,
    newPortNumber,
    newPortLabel,
    newPortProvider,
    onPortNumberChange,
    onPortLabelChange,
    onPortProviderChange,
    onAddPort,
    onRemovePort,
    currentView,
    onNavigateToView,
}: PortForwardingViewProps) {
    if (currentView === 'diagnostics') {
        return <CloudflareDiagnosticsView onBack={() => onNavigateToView('')} />;
    }

    // Per-port diagnostics: view = "port-diagnose-XXXX"
    const portDiagnoseMatch = currentView.match(/^port-diagnose-(\d+)$/);
    if (portDiagnoseMatch) {
        const diagPort = parseInt(portDiagnoseMatch[1], 10);
        const portData = ports.find(p => p.localPort === diagPort);
        return <PortDiagnoseView port={diagPort} portData={portData} onBack={() => onNavigateToView('')} />;
    }

    return (
        <div className="mcc-ports">
            <div className="mcc-section-header">
                <h2>Port Forwarding</h2>
            </div>
            <CloudflareStatusBanner onClick={() => onNavigateToView('diagnostics')} />
            {error && (
                <div className="mcc-ports-error">Error: {error}</div>
            )}
            {actionError && (
                <div className="mcc-ports-error">{actionError}</div>
            )}
            <div className="mcc-ports-subtitle">
                {loading ? 'Loading...' : `Active Forwards (${ports.length})`}
            </div>
            <div className="mcc-ports-list">
                {ports.map(port => (
                    <PortForwardCard key={port.localPort} port={port} onRemove={() => onRemovePort(port.localPort)} onNavigateToView={onNavigateToView} />
                ))}
                {!loading && ports.length === 0 && (
                    <div className="mcc-ports-empty">No port forwards active. Add one below.</div>
                )}
            </div>
            <div className="mcc-add-port-section">
                {showNewForm ? (
                    <div className="mcc-add-port-form">
                        <div className="mcc-add-port-header">
                            <span>Add Port Forward</span>
                            <button className="mcc-close-btn" onClick={onToggleNewForm}>×</button>
                        </div>
                        <div className="mcc-add-port-fields">
                            <div className="mcc-form-field">
                                <label>Port</label>
                                <input
                                    type="number"
                                    placeholder="8080"
                                    value={newPortNumber}
                                    onChange={e => onPortNumberChange(e.target.value)}
                                />
                            </div>
                            <div className="mcc-form-field">
                                <label>Label</label>
                                <input
                                    type="text"
                                    placeholder="My Service"
                                    value={newPortLabel}
                                    onChange={e => onPortLabelChange(e.target.value)}
                                />
                            </div>
                        </div>
                        <div className="mcc-form-field mcc-provider-field">
                            <label>Provider</label>
                            <div className="mcc-provider-options">
                                {availableProviders.filter(p => p.available).map(p => (
                                    <button
                                        key={p.id}
                                        className={`mcc-provider-btn ${newPortProvider === p.id ? 'active' : ''}`}
                                        onClick={() => onPortProviderChange(p.id as TunnelProvider)}
                                        title={p.description}
                                    >
                                        {p.name}
                                    </button>
                                ))}
                            </div>
                        </div>
                        <button className="mcc-forward-btn" onClick={onAddPort}>
                            Forward
                        </button>
                    </div>
                ) : (
                    <button className="mcc-add-port-btn" onClick={onToggleNewForm}>
                        <PlusIcon />
                        <span>Add Port Forward</span>
                    </button>
                )}
            </div>
            <PortForwardingHelp />
        </div>
    );
}

// Help section explaining how port forwarding works
function PortForwardingHelp() {
    const [expanded, setExpanded] = useState(false);

    return (
        <div className="mcc-ports-help">
            <button className="mcc-ports-help-toggle" onClick={() => setExpanded(!expanded)}>
                <span className="mcc-ports-help-icon">?</span>
                <span>How does port forwarding work?</span>
                <span className={`mcc-ports-help-chevron ${expanded ? 'expanded' : ''}`}>›</span>
            </button>
            {expanded && (
                <div className="mcc-ports-help-content">
                    <p>
                        Port forwarding creates a secure public URL for a service running on a local port
                        of this machine, making it accessible from anywhere on the internet.
                    </p>
                    <div className="mcc-ports-help-steps">
                        <div className="mcc-ports-help-step">
                            <span className="mcc-ports-help-step-num">1</span>
                            <span>You specify a local port (e.g. <code>3000</code>) where your app is running.</span>
                        </div>
                        <div className="mcc-ports-help-step">
                            <span className="mcc-ports-help-step-num">2</span>
                            <span>The server starts a tunnel process to create a public URL.</span>
                        </div>
                        <div className="mcc-ports-help-step">
                            <span className="mcc-ports-help-step-num">3</span>
                            <span>A temporary public URL is assigned that proxies traffic to your local service.</span>
                        </div>
                    </div>

                    <p><strong>Providers:</strong></p>
                    <div className="mcc-ports-help-provider">
                        <strong>localtunnel</strong> (default)
                        <div className="mcc-ports-help-cmd">
                            <code>npx localtunnel --port PORT</code>
                        </div>
                        <span>Free, no account needed. URL: <code>https://xxx.loca.lt</code></span>
                    </div>
                    <div className="mcc-ports-help-provider">
                        <strong>Cloudflare Quick Tunnel</strong>
                        <div className="mcc-ports-help-cmd">
                            <code>cloudflared tunnel --url http://127.0.0.1:PORT</code>
                        </div>
                        <span>Free via Cloudflare Quick Tunnels. URL: <code>https://xxx.trycloudflare.com</code></span>
                    </div>
                    <div className="mcc-ports-help-provider">
                        <strong>Cloudflare Named Tunnel</strong>
                        <div className="mcc-ports-help-cmd">
                            <code>cloudflared tunnel route dns TUNNEL random-words.YOUR-DOMAIN</code>
                        </div>
                        <span>Uses your own domain with a named Cloudflare tunnel. A random subdomain (e.g. <code>brave-lake-fern.your-domain.xyz</code>) is generated for each port to prevent guessing. Requires <code>cloudflared</code> setup and <code>base_domain</code> in the server config file.</span>
                    </div>

                    <p className="mcc-ports-help-note">
                        <strong>Note:</strong> localtunnel and Cloudflare Quick tunnels are temporary (URLs change each time, no account needed). Named Cloudflare tunnels use random subdomains on your own domain for security, and require a Cloudflare account with tunnel setup.
                    </p>
                </div>
            )}
        </div>
    );
}

// --- Cloudflare Diagnostics ---

function CloudflareStatusBanner({ onClick }: { onClick: () => void }) {
    const [data, setData] = useState<DiagnosticsData | null>(null);

    useEffect(() => {
        apiFetchDiagnostics()
            .then(d => setData(d))
            .catch(() => {});
    }, []);

    const statusIcon = !data ? '⏳' : data.overall === 'ok' ? '✅' : data.overall === 'warning' ? '⚠️' : '❌';
    const statusText = !data ? 'Checking...' : data.overall === 'ok' ? 'Cloudflare: Ready' : data.overall === 'warning' ? 'Cloudflare: Warning' : 'Cloudflare: Issues Found';

    return (
        <button className={`mcc-cf-status-banner mcc-cf-status-${data?.overall ?? 'loading'}`} onClick={onClick}>
            <span className="mcc-cf-status-icon">{statusIcon}</span>
            <span className="mcc-cf-status-text">{statusText}</span>
            <span className="mcc-cf-status-chevron">›</span>
        </button>
    );
}

function CloudflareDiagnosticsView({ onBack }: { onBack: () => void }) {
    const [data, setData] = useState<DiagnosticsData | null>(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        setLoading(true);
        apiFetchDiagnostics()
            .then(d => { setData(d); setLoading(false); })
            .catch(() => setLoading(false));
    }, []);

    const statusColors: Record<string, string> = {
        ok: '#22c55e',
        warning: '#eab308',
        error: '#ef4444',
    };

    return (
        <div className="mcc-ports">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={onBack}>← Back</button>
                <h2>Cloudflare Diagnostics</h2>
            </div>
            {loading ? (
                <div className="mcc-diag-loading">Running diagnostics...</div>
            ) : !data ? (
                <div className="mcc-ports-error">Failed to load diagnostics</div>
            ) : (
                <>
                    <div className={`mcc-diag-overall mcc-diag-overall-${data.overall}`}>
                        <span className="mcc-diag-overall-icon">
                            {data.overall === 'ok' ? '✅' : data.overall === 'warning' ? '⚠️' : '❌'}
                        </span>
                        <span>
                            {data.overall === 'ok' ? 'All checks passed' :
                             data.overall === 'warning' ? 'Some warnings' :
                             'Issues found'}
                        </span>
                    </div>
                    <div className="mcc-diag-checks">
                        {data.checks.map(check => (
                            <div key={check.id} className="mcc-diag-check">
                                <div className="mcc-diag-check-header">
                                    <span
                                        className="mcc-diag-check-dot"
                                        style={{ background: statusColors[check.status] ?? '#64748b' }}
                                    />
                                    <span className="mcc-diag-check-label">{check.label}</span>
                                    <span className={`mcc-diag-check-status mcc-diag-check-status-${check.status}`}>
                                        {check.status.toUpperCase()}
                                    </span>
                                </div>
                                <div className="mcc-diag-check-desc">{check.description}</div>
                            </div>
                        ))}
                    </div>
                    <button className="mcc-diag-refresh" onClick={() => {
                        setLoading(true);
                        apiFetchDiagnostics()
                            .then(d => { setData(d); setLoading(false); })
                            .catch(() => setLoading(false));
                    }}>
                        Refresh
                    </button>
                </>
            )}
        </div>
    );
}

// Per-Port Diagnose View
function PortDiagnoseView({ port, portData, onBack }: { port: number; portData?: PortForward; onBack: () => void }) {
    const [result, setResult] = useState<{ status: string; body: string } | null>(null);
    const [loading, setLoading] = useState(false);

    const publicUrl = portData?.publicUrl;

    const runCheck = async () => {
        if (!publicUrl) return;
        setLoading(true);
        try {
            const resp = await fetch(publicUrl, { mode: 'no-cors' }).catch(() => null);
            if (!resp) {
                setResult({ status: 'error', body: 'Failed to reach the URL. The tunnel may not be active or the local service is not running.' });
            } else {
                // no-cors means we can't read the body, but we can check if it succeeded
                setResult({ status: 'reachable', body: `Got response (opaque due to CORS). Status type: ${resp.type}` });
            }
        } catch {
            setResult({ status: 'error', body: 'Network error when trying to reach the URL.' });
        }
        setLoading(false);
    };

    // Detect common issues from logs
    const [logs, setLogs] = useState<string[]>([]);
    useEffect(() => {
        apiFetchPortLogs(port)
            .then(data => setLogs(data))
            .catch(() => {});
    }, [port]);

    const logsText = logs.join('\n');
    const isViteError = logsText.includes('allowedHosts') || logsText.includes('This host') || logsText.includes('is not allowed');
    const isTunnelError = logsText.includes('failed to start') || logsText.includes('tunnel exited');
    const isTimeout = logsText.includes('timeout');

    return (
        <div className="mcc-ports">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={onBack}>← Back</button>
                <h2>Diagnose Port {port}</h2>
            </div>

            {portData && (
                <div className="mcc-diag-port-info">
                    <div><strong>Status:</strong> {portData.status}</div>
                    <div><strong>Provider:</strong> {portData.provider}</div>
                    {publicUrl && <div><strong>URL:</strong> <a href={publicUrl} target="_blank" rel="noopener noreferrer">{publicUrl}</a></div>}
                    {portData.error && <div className="mcc-port-url-error"><strong>Error:</strong> {portData.error}</div>}
                </div>
            )}

            <div className="mcc-diag-checks">
                {isViteError && (
                    <div className="mcc-diag-check">
                        <div className="mcc-diag-check-header">
                            <span className="mcc-diag-check-dot" style={{ background: '#ef4444' }} />
                            <span className="mcc-diag-check-label">Vite Host Blocked</span>
                            <span className="mcc-diag-check-status mcc-diag-check-status-error">DETECTED</span>
                        </div>
                        <div className="mcc-diag-check-desc">
                            Vite's dev server is blocking requests from the tunnel hostname. Add the following to your <code>vite.config.ts</code>:
                        </div>
                        <div className="mcc-ports-help-cmd" style={{ margin: '8px 0' }}>
                            <code>{`server: {\n  allowedHosts: true\n}`}</code>
                        </div>
                        <div className="mcc-diag-check-desc">
                            Then restart the Vite dev server.
                        </div>
                    </div>
                )}

                {isTunnelError && (
                    <div className="mcc-diag-check">
                        <div className="mcc-diag-check-header">
                            <span className="mcc-diag-check-dot" style={{ background: '#ef4444' }} />
                            <span className="mcc-diag-check-label">Tunnel Process Error</span>
                            <span className="mcc-diag-check-status mcc-diag-check-status-error">DETECTED</span>
                        </div>
                        <div className="mcc-diag-check-desc">
                            The tunnel process failed to start or exited unexpectedly. Check the logs below for details.
                        </div>
                    </div>
                )}

                {isTimeout && (
                    <div className="mcc-diag-check">
                        <div className="mcc-diag-check-header">
                            <span className="mcc-diag-check-dot" style={{ background: '#eab308' }} />
                            <span className="mcc-diag-check-label">Timeout</span>
                            <span className="mcc-diag-check-status mcc-diag-check-status-warning">DETECTED</span>
                        </div>
                        <div className="mcc-diag-check-desc">
                            The tunnel took too long to establish. This may be a network issue.
                        </div>
                    </div>
                )}

                {!isViteError && !isTunnelError && !isTimeout && portData?.status === PortStatuses.Active && (
                    <div className="mcc-diag-check">
                        <div className="mcc-diag-check-header">
                            <span className="mcc-diag-check-dot" style={{ background: '#22c55e' }} />
                            <span className="mcc-diag-check-label">No issues detected</span>
                            <span className="mcc-diag-check-status mcc-diag-check-status-ok">OK</span>
                        </div>
                        <div className="mcc-diag-check-desc">
                            The tunnel appears to be running normally. If you're having issues, check that the local service on port {port} is running.
                        </div>
                    </div>
                )}
            </div>

            {publicUrl && (
                <button className="mcc-diag-refresh" onClick={runCheck} disabled={loading}>
                    {loading ? 'Checking...' : 'Test Connectivity'}
                </button>
            )}
            {result && (
                <div className={`mcc-diag-port-info ${result.status === 'error' ? 'mcc-diag-port-error' : ''}`}>
                    {result.body}
                </div>
            )}

            {logs.length > 0 && (
                <>
                    <div className="mcc-ports-subtitle" style={{ margin: '16px 16px 8px' }}>Recent Logs</div>
                    <div style={{ margin: '0 16px 16px' }}>
                        <LogViewer lines={logs.map(text => ({ text }))} />
                    </div>
                </>
            )}
        </div>
    );
}

// Port Forward Card
interface PortForwardCardProps {
    port: PortForward;
    onRemove: () => void;
    onNavigateToView: (view: string) => void;
}

function PortForwardCard({ port, onRemove, onNavigateToView }: PortForwardCardProps) {
    const [showLogs, setShowLogs] = useState(false);
    const [logs, setLogs] = useState<string[]>([]);
    const [copied, setCopied] = useState(false);

    const statusIcon = port.status === PortStatuses.Active ? '🟢' :
                       port.status === PortStatuses.Connecting ? '🟡' : '🔴';

    const handleCopy = () => {
        if (port.publicUrl) {
            navigator.clipboard.writeText(port.publicUrl);
            setCopied(true);
            setTimeout(() => setCopied(false), 1500);
        }
    };

    // Fetch logs when visible
    useEffect(() => {
        if (!showLogs) return;

        const fetchLogs = async () => {
            try {
                const data = await apiFetchPortLogs(port.localPort);
                setLogs(data);
            } catch { /* ignore */ }
        };

        fetchLogs();
        const timer = setInterval(fetchLogs, 2000);
        return () => clearInterval(timer);
    }, [showLogs, port.localPort]);

    return (
        <div className="mcc-port-card">
            <div className="mcc-port-header">
                <span className="mcc-port-status">{statusIcon}</span>
                <span className="mcc-port-number">:{port.localPort}</span>
                <span className="mcc-port-arrow">→</span>
                <span className="mcc-port-label">{port.label}</span>
                <span className="mcc-port-provider-badge">{port.provider}</span>
            </div>
            {port.publicUrl ? (
                <div className="mcc-port-url">
                    <a href={port.publicUrl} target="_blank" rel="noopener noreferrer" className="mcc-port-url-link">
                        {port.publicUrl}
                    </a>
                    <button className="mcc-port-copy-icon" onClick={handleCopy} title="Copy URL">
                        {copied ? '✓' : '📋'}
                    </button>
                </div>
            ) : port.status === PortStatuses.Connecting ? (
                <div className="mcc-port-url mcc-port-url-connecting">Establishing tunnel...</div>
            ) : port.error ? (
                <div className="mcc-port-url mcc-port-url-error">{port.error}</div>
            ) : null}
            <div className="mcc-port-actions">
                <button
                    className={`mcc-port-action-btn mcc-port-logs-btn ${showLogs ? 'active' : ''}`}
                    onClick={() => setShowLogs(!showLogs)}
                >
                    Logs
                </button>
                <button
                    className="mcc-port-action-btn"
                    onClick={() => onNavigateToView(`port-diagnose-${port.localPort}`)}
                >
                    Diagnose
                </button>
                <button className="mcc-port-action-btn mcc-port-stop" onClick={onRemove}>Stop</button>
            </div>
            {showLogs && (
                <LogViewer
                    lines={logs.map(text => ({ text }))}
                    className="mcc-port-logs-margin"
                />
            )}
        </div>
    );
}


// Navigation Button
interface NavButtonProps {
    icon: React.ReactNode;
    label: string;
    active: boolean;
    onClick: () => void;
}

function NavButton({ icon, label, active, onClick }: NavButtonProps) {
    return (
        <button className={`mcc-nav-btn ${active ? 'active' : ''}`} onClick={onClick}>
            {icon}
            <span>{label}</span>
        </button>
    );
}

// Icons
function MenuIcon() {
    return (
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <line x1="3" y1="12" x2="21" y2="12" />
            <line x1="3" y1="6" x2="21" y2="6" />
            <line x1="3" y1="18" x2="21" y2="18" />
        </svg>
    );
}

function SettingsIcon() {
    return (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <circle cx="12" cy="12" r="3" />
            <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" />
        </svg>
    );
}

function ProfileIcon() {
    return (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
            <circle cx="12" cy="7" r="4" />
        </svg>
    );
}

function HomeIcon() {
    return (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />
            <polyline points="9 22 9 12 15 12 15 22" />
        </svg>
    );
}

function AgentIcon() {
    return (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <rect x="3" y="11" width="18" height="10" rx="2" />
            <circle cx="12" cy="5" r="2" />
            <path d="M12 7v4" />
            <line x1="8" y1="16" x2="8" y2="16" />
            <line x1="16" y1="16" x2="16" y2="16" />
        </svg>
    );
}

function TerminalIcon() {
    return (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <polyline points="4 17 10 11 4 5" />
            <line x1="12" y1="19" x2="20" y2="19" />
        </svg>
    );
}

function PortsIcon() {
    return (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71" />
            <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" />
        </svg>
    );
}

function FilesIcon() {
    return (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" />
        </svg>
    );
}

function PlusIcon() {
    return (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <line x1="12" y1="5" x2="12" y2="19" />
            <line x1="5" y1="12" x2="19" y2="12" />
        </svg>
    );
}
