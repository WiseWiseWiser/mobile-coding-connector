import { useState, useEffect } from 'react';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { useCurrent } from '../../hooks/useCurrent';
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

// Port forward status
const PortStatuses = {
    Active: 'active',
    Connecting: 'connecting',
    Error: 'error',
} as const;

type PortStatus = typeof PortStatuses[keyof typeof PortStatuses];

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

interface PortForward {
    localPort: number;
    label: string;
    publicUrl: string;
    status: PortStatus;
}

interface TerminalLine {
    type: 'input' | 'output';
    content: string;
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

const mockPortForwards: PortForward[] = [
    { localPort: 5173, label: 'Frontend Dev', publicUrl: 'https://abc123.tunnel.dev', status: PortStatuses.Active },
    { localPort: 3000, label: 'API Server', publicUrl: 'https://xyz789.tunnel.dev', status: PortStatuses.Active },
];

const mockTerminalLines: TerminalLine[] = [
    { type: 'input', content: 'npm run dev' },
    { type: 'output', content: '' },
    { type: 'output', content: '> my-react-app@0.1.0 dev' },
    { type: 'output', content: '> vite' },
    { type: 'output', content: '' },
    { type: 'output', content: '  VITE v5.0.0  ready in 234 ms' },
    { type: 'output', content: '' },
    { type: 'output', content: '  ➜  Local:   http://localhost:5173/' },
    { type: 'output', content: '  ➜  Network: http://192.168.1.5:5173/' },
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
    const [terminalInput, setTerminalInput] = useState('');
    const [agentStatus, setAgentStatus] = useState<AgentStatus>(AgentStatuses.Executing);
    const [showNewPortForm, setShowNewPortForm] = useState(false);
    const [newPortNumber, setNewPortNumber] = useState('');
    const [newPortLabel, setNewPortLabel] = useState('');

    // Refs for callbacks
    const activeTabRef = useCurrent(activeTab);
    const currentWorkspaceRef = useCurrent(currentWorkspace);

    // Sync URL with state changes
    useEffect(() => {
        const params = new URLSearchParams();
        if (activeTab !== NavTabs.Home) {
            params.set('tab', activeTab);
        }
        setSearchParams(params, { replace: true });
    }, [activeTab, setSearchParams]);

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
        // Navigate to workspace URL
        navigate(`/mockups/v2/${workspace.id}?tab=agent`);
    };

    const handleTabChange = (tab: NavTab) => {
        setActiveTab(tab);
        if (currentWorkspace) {
            navigate(`/mockups/v2/${currentWorkspace.id}?tab=${tab}`, { replace: true });
        } else {
            navigate(`/mockups/v2?tab=${tab}`, { replace: true });
        }
    };

    const handleSendPrompt = () => {
        if (!chatInput.trim()) return;
        // In real app, this would send to server
        setChatInput('');
        setAgentStatus(AgentStatuses.Thinking);
    };

    const handleTerminalSubmit = () => {
        if (!terminalInput.trim()) return;
        // In real app, this would send to terminal
        setTerminalInput('');
    };

    const handleAddPortForward = () => {
        if (!newPortNumber || !newPortLabel) return;
        // In real app, this would create port forward
        setShowNewPortForm(false);
        setNewPortNumber('');
        setNewPortLabel('');
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
                return (
                    <TerminalView
                        lines={mockTerminalLines}
                        inputValue={terminalInput}
                        onInputChange={setTerminalInput}
                        onSubmit={handleTerminalSubmit}
                    />
                );
            case NavTabs.Ports:
                return (
                    <PortForwardingView
                        ports={mockPortForwards}
                        showNewForm={showNewPortForm}
                        onToggleNewForm={() => setShowNewPortForm(!showNewPortForm)}
                        newPortNumber={newPortNumber}
                        newPortLabel={newPortLabel}
                        onPortNumberChange={setNewPortNumber}
                        onPortLabelChange={setNewPortLabel}
                        onAddPort={handleAddPortForward}
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
                {renderContent()}
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

// Terminal View
interface TerminalViewProps {
    lines: TerminalLine[];
    inputValue: string;
    onInputChange: (value: string) => void;
    onSubmit: () => void;
}

function TerminalView({ lines, inputValue, onInputChange, onSubmit }: TerminalViewProps) {
    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            onSubmit();
        }
    };

    return (
        <div className="mcc-terminal">
            <div className="mcc-terminal-header">
                <div className="mcc-terminal-tabs">
                    <button className="mcc-terminal-tab active">Tab 1</button>
                    <button className="mcc-terminal-tab">Tab 2</button>
                    <button className="mcc-terminal-tab-add">+</button>
                </div>
            </div>
            <div className="mcc-terminal-output">
                <div className="mcc-terminal-line mcc-terminal-prompt">
                    ~/my-react-app $
                </div>
                {lines.map((line, idx) => (
                    <div
                        key={idx}
                        className={`mcc-terminal-line ${line.type === 'input' ? 'mcc-terminal-input' : ''}`}
                    >
                        {line.content}
                    </div>
                ))}
                <div className="mcc-terminal-line mcc-terminal-prompt">
                    ~/my-react-app $ <span className="mcc-cursor">_</span>
                </div>
            </div>
            <div className="mcc-terminal-input-area">
                <input
                    type="text"
                    className="mcc-terminal-input-field"
                    placeholder="Enter command..."
                    value={inputValue}
                    onChange={e => onInputChange(e.target.value)}
                    onKeyDown={handleKeyDown}
                />
                <button className="mcc-terminal-send-btn" onClick={onSubmit}>
                    <SendIcon />
                </button>
            </div>
            <div className="mcc-terminal-shortcuts">
                <button className="mcc-shortcut-btn">Tab</button>
                <button className="mcc-shortcut-btn">Ctrl</button>
                <button className="mcc-shortcut-btn">↑</button>
                <button className="mcc-shortcut-btn">↓</button>
                <button className="mcc-shortcut-btn">C</button>
                <button className="mcc-shortcut-btn">D</button>
                <button className="mcc-shortcut-btn">L</button>
            </div>
        </div>
    );
}

// Port Forwarding View
interface PortForwardingViewProps {
    ports: PortForward[];
    showNewForm: boolean;
    onToggleNewForm: () => void;
    newPortNumber: string;
    newPortLabel: string;
    onPortNumberChange: (value: string) => void;
    onPortLabelChange: (value: string) => void;
    onAddPort: () => void;
}

function PortForwardingView({
    ports,
    showNewForm,
    onToggleNewForm,
    newPortNumber,
    newPortLabel,
    onPortNumberChange,
    onPortLabelChange,
    onAddPort,
}: PortForwardingViewProps) {
    return (
        <div className="mcc-ports">
            <div className="mcc-section-header">
                <h2>Port Forwarding</h2>
            </div>
            <div className="mcc-ports-subtitle">Active Forwards</div>
            <div className="mcc-ports-list">
                {ports.map(port => (
                    <PortForwardCard key={port.localPort} port={port} />
                ))}
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
        </div>
    );
}

// Port Forward Card
interface PortForwardCardProps {
    port: PortForward;
}

function PortForwardCard({ port }: PortForwardCardProps) {
    const statusIcon = port.status === PortStatuses.Active ? '🟢' :
                       port.status === PortStatuses.Connecting ? '🟡' : '🔴';

    return (
        <div className="mcc-port-card">
            <div className="mcc-port-header">
                <span className="mcc-port-status">{statusIcon}</span>
                <span className="mcc-port-number">:{port.localPort}</span>
                <span className="mcc-port-arrow">→</span>
                <span className="mcc-port-label">{port.label}</span>
            </div>
            <div className="mcc-port-url">{port.publicUrl}</div>
            <div className="mcc-port-actions">
                <button className="mcc-port-action-btn">Copy</button>
                <button className="mcc-port-action-btn">Open</button>
                <button className="mcc-port-action-btn mcc-port-stop">Stop</button>
            </div>
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
