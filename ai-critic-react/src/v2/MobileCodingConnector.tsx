import { useState, useEffect, useRef } from 'react';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { useCurrent } from '../hooks/useCurrent';
import { usePortForwards } from '../hooks/usePortForwards';
import type { PortForward, TunnelProvider, ProviderInfo } from '../hooks/usePortForwards';
import { PortStatuses, TunnelProviders } from '../hooks/usePortForwards';
import { TerminalManager } from './TerminalManager';
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

// Workspace status
const WorkspaceStatuses = {
    Running: 'running',
    Stopped: 'stopped',
    Error: 'error',
} as const;

type WorkspaceStatus = typeof WorkspaceStatuses[keyof typeof WorkspaceStatuses];

// Agent status
const AgentStatuses = {
    Idle: 'idle',
    Thinking: 'thinking',
    Executing: 'executing',
} as const;

type AgentStatus = typeof AgentStatuses[keyof typeof AgentStatuses];


// Types
interface Workspace {
    id: string;
    name: string;
    type: string;
    status: WorkspaceStatus;
    lastAccessed: string;
    memory: string;
}

interface ChatMessage {
    id: string;
    role: 'user' | 'agent';
    content: string;
    actions?: AgentAction[];
}

interface AgentAction {
    type: string;
    status: 'pending' | 'running' | 'done' | 'error';
    description: string;
}

// Mock data
const mockWorkspaces: Workspace[] = [
    { id: '1', name: 'my-react-app', type: 'React', status: WorkspaceStatuses.Running, lastAccessed: '2h ago', memory: '512MB' },
    { id: '2', name: 'backend-api', type: 'Go', status: WorkspaceStatuses.Running, lastAccessed: '1d ago', memory: '256MB' },
    { id: '3', name: 'ml-training', type: 'Python', status: WorkspaceStatuses.Stopped, lastAccessed: '3d ago', memory: '--' },
];

const mockChatHistory: ChatMessage[] = [
    { id: '1', role: 'user', content: 'Add a login page with Google OAuth' },
    {
        id: '2',
        role: 'agent',
        content: "I'll create a login page with Google OAuth integration. Let me set that up for you.",
        actions: [
            { type: 'file_create', status: 'done', description: 'Created LoginPage.tsx' },
            { type: 'file_edit', status: 'done', description: 'Added OAuth config' },
            { type: 'install', status: 'running', description: 'Installing dependencies...' },
        ],
    },
];

export function MobileCodingConnector() {
    const navigate = useNavigate();
    const { workspaceId } = useParams<{ workspaceId: string }>();
    const [searchParams, setSearchParams] = useSearchParams();
    
    // Get tab from URL search params, default to 'home'
    const tabFromUrl = (searchParams.get('tab') as NavTab) || NavTabs.Home;
    const [activeTab, setActiveTab] = useState<NavTab>(tabFromUrl);
    
    // Find workspace from URL param
    const workspaceFromUrl = workspaceId ? mockWorkspaces.find(w => w.id === workspaceId) ?? null : null;
    const [currentWorkspace, setCurrentWorkspace] = useState<Workspace | null>(workspaceFromUrl);
    
    const [chatInput, setChatInput] = useState('');
    const [agentStatus, setAgentStatus] = useState<AgentStatus>(AgentStatuses.Executing);

    // Port forwarding - real API
    const { ports, providers: availableProviders, loading: portsLoading, error: portsError, addPort, removePort, refresh: refreshPorts } = usePortForwards();
    const [showNewPortForm, setShowNewPortForm] = useState(false);
    const [newPortNumber, setNewPortNumber] = useState('');
    const [newPortLabel, setNewPortLabel] = useState('');
    const [newPortProvider, setNewPortProvider] = useState<TunnelProvider>(TunnelProviders.Localtunnel);
    const [portActionError, setPortActionError] = useState<string | null>(null);

    // Refs for callbacks
    const activeTabRef = useCurrent(activeTab);
    const currentWorkspaceRef = useCurrent(currentWorkspace);

    // Get sub-view from URL (e.g., view=diagnostics)
    const viewFromUrl = searchParams.get('view') || '';

    const navigateToView = (view: string) => {
        const params = new URLSearchParams(searchParams);
        if (view) {
            params.set('view', view);
        } else {
            params.delete('view');
        }
        setSearchParams(params, { replace: true });
    };

    // Sync URL with tab changes
    useEffect(() => {
        const params = new URLSearchParams(searchParams);
        if (activeTab !== NavTabs.Home) {
            params.set('tab', activeTab);
        } else {
            params.delete('tab');
        }
        // Clear view when switching tabs
        params.delete('view');
        setSearchParams(params, { replace: true });
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [activeTab]);

    // Sync state from URL on mount or URL change
    useEffect(() => {
        if (workspaceId) {
            const ws = mockWorkspaces.find(w => w.id === workspaceId);
            if (ws && ws.id !== currentWorkspaceRef.current?.id) {
                setCurrentWorkspace(ws);
            }
        }
        const urlTab = searchParams.get('tab') as NavTab;
        if (urlTab && urlTab !== activeTabRef.current) {
            setActiveTab(urlTab);
        }
    }, [workspaceId, searchParams, currentWorkspaceRef, activeTabRef]);

    const handleSelectWorkspace = (workspace: Workspace) => {
        setCurrentWorkspace(workspace);
        setActiveTab(NavTabs.Agent);
        // Navigate to workspace URL - using /v2 path
        navigate(`/v2/${workspace.id}?tab=agent`);
    };

    const handleTabChange = (tab: NavTab) => {
        setActiveTab(tab);
        if (currentWorkspace) {
            navigate(`/v2/${currentWorkspace.id}?tab=${tab}`, { replace: true });
        } else {
            navigate(`/v2?tab=${tab}`, { replace: true });
        }
    };

    const handleSendPrompt = () => {
        if (!chatInput.trim()) return;
        // In real app, this would send to server
        setChatInput('');
        setAgentStatus(AgentStatuses.Thinking);
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
                return <WorkspaceListView workspaces={mockWorkspaces} onSelect={handleSelectWorkspace} />;
            case NavTabs.Agent:
                return (
                    <AgentChatView
                        workspace={currentWorkspace}
                        messages={mockChatHistory}
                        agentStatus={agentStatus}
                        inputValue={chatInput}
                        onInputChange={setChatInput}
                        onSend={handleSendPrompt}
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
                return <FilesView />;
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
                    {currentWorkspace ? currentWorkspace.name : 'Mobile Coding Connector'}
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
                    <TerminalManager isVisible={true} />
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

// Workspace List View
interface WorkspaceListViewProps {
    workspaces: Workspace[];
    onSelect: (workspace: Workspace) => void;
}

function WorkspaceListView({ workspaces, onSelect }: WorkspaceListViewProps) {
    return (
        <div className="mcc-workspace-list">
            <div className="mcc-section-header">
                <h2>Your Workspaces</h2>
            </div>
            <div className="mcc-workspace-cards">
                {workspaces.map(workspace => (
                    <WorkspaceCard key={workspace.id} workspace={workspace} onClick={() => onSelect(workspace)} />
                ))}
            </div>
            <button className="mcc-new-workspace-btn">
                <PlusIcon />
                <span>New Workspace</span>
            </button>
        </div>
    );
}

// Workspace Card
interface WorkspaceCardProps {
    workspace: Workspace;
    onClick: () => void;
}

function WorkspaceCard({ workspace, onClick }: WorkspaceCardProps) {
    const statusClass = `mcc-status-${workspace.status}`;
    const statusIcon = workspace.status === WorkspaceStatuses.Running ? '🟢' :
                       workspace.status === WorkspaceStatuses.Stopped ? '🔴' : '🟡';

    return (
        <div className={`mcc-workspace-card ${statusClass}`} onClick={onClick}>
            <div className="mcc-workspace-card-header">
                <span className="mcc-workspace-status-icon">{statusIcon}</span>
                <span className="mcc-workspace-name">{workspace.name}</span>
            </div>
            <div className="mcc-workspace-card-meta">
                <span>{workspace.type}</span>
                <span>•</span>
                <span>{workspace.lastAccessed}</span>
                <span>•</span>
                <span>{workspace.memory}</span>
            </div>
        </div>
    );
}

// Agent Chat View
interface AgentChatViewProps {
    workspace: Workspace | null;
    messages: ChatMessage[];
    agentStatus: AgentStatus;
    inputValue: string;
    onInputChange: (value: string) => void;
    onSend: () => void;
}

function AgentChatView({ workspace, messages, agentStatus, inputValue, onInputChange, onSend }: AgentChatViewProps) {
    if (!workspace) {
        return (
            <div className="mcc-empty-state">
                <AgentIcon />
                <h3>No Workspace Selected</h3>
                <p>Select a workspace from the Home tab to start chatting with the agent.</p>
            </div>
        );
    }

    return (
        <div className="mcc-agent-chat">
            <div className="mcc-chat-header">
                <span className="mcc-chat-context">Context: {workspace.name}</span>
                {agentStatus !== AgentStatuses.Idle && (
                    <span className={`mcc-agent-status mcc-agent-${agentStatus}`}>
                        {agentStatus === AgentStatuses.Thinking ? '🤔 Thinking...' : '⚡ Executing...'}
                    </span>
                )}
            </div>
            <div className="mcc-chat-messages">
                {messages.map(message => (
                    <ChatMessageItem key={message.id} message={message} />
                ))}
            </div>
            <div className="mcc-chat-input-area">
                <textarea
                    className="mcc-chat-input"
                    placeholder="Type your prompt..."
                    value={inputValue}
                    onChange={e => onInputChange(e.target.value)}
                    rows={2}
                />
                <button className="mcc-send-btn" onClick={onSend}>
                    <SendIcon />
                </button>
            </div>
        </div>
    );
}

// Chat Message Item
interface ChatMessageItemProps {
    message: ChatMessage;
}

function ChatMessageItem({ message }: ChatMessageItemProps) {
    const isUser = message.role === 'user';

    return (
        <div className={`mcc-chat-message ${isUser ? 'mcc-message-user' : 'mcc-message-agent'}`}>
            <div className="mcc-message-avatar">
                {isUser ? '👤' : '🤖'}
            </div>
            <div className="mcc-message-content">
                <p>{message.content}</p>
                {message.actions && message.actions.length > 0 && (
                    <div className="mcc-message-actions">
                        {message.actions.map((action, idx) => (
                            <div key={idx} className={`mcc-action-item mcc-action-${action.status}`}>
                                <span className="mcc-action-icon">
                                    {action.status === 'done' ? '✓' : action.status === 'running' ? '○' : '•'}
                                </span>
                                <span>{action.description}</span>
                            </div>
                        ))}
                    </div>
                )}
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

interface DiagnosticCheck {
    id: string;
    label: string;
    status: 'ok' | 'warning' | 'error';
    description: string;
}

interface DiagnosticsData {
    overall: 'ok' | 'warning' | 'error';
    checks: DiagnosticCheck[];
}

function CloudflareStatusBanner({ onClick }: { onClick: () => void }) {
    const [data, setData] = useState<DiagnosticsData | null>(null);

    useEffect(() => {
        fetch('/api/ports/diagnostics')
            .then(r => r.json())
            .then((d: DiagnosticsData) => setData(d))
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
        fetch('/api/ports/diagnostics')
            .then(r => r.json())
            .then((d: DiagnosticsData) => {
                setData(d);
                setLoading(false);
            })
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
                        fetch('/api/ports/diagnostics')
                            .then(r => r.json())
                            .then((d: DiagnosticsData) => { setData(d); setLoading(false); })
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
        fetch(`/api/ports/logs?port=${port}`)
            .then(r => r.json())
            .then((data: string[]) => setLogs(data ?? []))
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
                    <div className="mcc-port-logs" style={{ margin: '0 16px 16px' }}>
                        {logs.map((line, i) => (
                            <div key={i} className="mcc-port-log-line">{line}</div>
                        ))}
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
    const logsContainerRef = useRef<HTMLDivElement>(null);
    const isAtBottomRef = useRef(true);

    const statusIcon = port.status === PortStatuses.Active ? '🟢' :
                       port.status === PortStatuses.Connecting ? '🟡' : '🔴';

    const handleCopy = () => {
        if (port.publicUrl) {
            navigator.clipboard.writeText(port.publicUrl);
            setCopied(true);
            setTimeout(() => setCopied(false), 1500);
        }
    };

    // Track whether user is at the bottom of the logs
    const handleLogsScroll = () => {
        const el = logsContainerRef.current;
        if (!el) return;
        isAtBottomRef.current = el.scrollHeight - el.scrollTop - el.clientHeight < 20;
    };

    // Fetch logs when visible
    useEffect(() => {
        if (!showLogs) return;

        const fetchLogs = async () => {
            try {
                const resp = await fetch(`/api/ports/logs?port=${port.localPort}`);
                if (resp.ok) {
                    const data = await resp.json();
                    setLogs(data ?? []);
                }
            } catch { /* ignore */ }
        };

        fetchLogs();
        const timer = setInterval(fetchLogs, 2000);
        return () => clearInterval(timer);
    }, [showLogs, port.localPort]);

    // Auto-scroll logs to bottom only if user is already at the bottom
    useEffect(() => {
        if (showLogs && isAtBottomRef.current && logsContainerRef.current) {
            logsContainerRef.current.scrollTop = logsContainerRef.current.scrollHeight;
        }
    }, [logs, showLogs]);

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
                <div className="mcc-port-logs" ref={logsContainerRef} onScroll={handleLogsScroll}>
                    {logs.length === 0 ? (
                        <div className="mcc-port-logs-empty">No logs yet...</div>
                    ) : (
                        logs.map((line, i) => (
                            <div key={i} className="mcc-port-log-line">{line}</div>
                        ))
                    )}
                </div>
            )}
        </div>
    );
}

// Files View (placeholder)
function FilesView() {
    return (
        <div className="mcc-files">
            <div className="mcc-section-header">
                <h2>Files</h2>
            </div>
            <div className="mcc-files-tree">
                <div className="mcc-file-item mcc-folder">
                    <span className="mcc-file-icon">📁</span>
                    <span>src</span>
                </div>
                <div className="mcc-file-item mcc-folder" style={{ paddingLeft: '24px' }}>
                    <span className="mcc-file-icon">📁</span>
                    <span>components</span>
                </div>
                <div className="mcc-file-item" style={{ paddingLeft: '48px' }}>
                    <span className="mcc-file-icon">📄</span>
                    <span>App.tsx</span>
                </div>
                <div className="mcc-file-item" style={{ paddingLeft: '48px' }}>
                    <span className="mcc-file-icon">📄</span>
                    <span>LoginPage.tsx</span>
                </div>
                <div className="mcc-file-item" style={{ paddingLeft: '24px' }}>
                    <span className="mcc-file-icon">📄</span>
                    <span>main.tsx</span>
                </div>
                <div className="mcc-file-item">
                    <span className="mcc-file-icon">📄</span>
                    <span>package.json</span>
                </div>
                <div className="mcc-file-item">
                    <span className="mcc-file-icon">📄</span>
                    <span>vite.config.ts</span>
                </div>
            </div>
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

function SendIcon() {
    return (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <line x1="22" y1="2" x2="11" y2="13" />
            <polygon points="22 2 15 22 11 13 2 9 22 2" />
        </svg>
    );
}
