import type { AgentDef, AgentSessionInfo, ExternalOpencodeSession } from '../../../api/agents';
import { SettingsIcon } from '../../icons';
import { SessionsSection, type SessionItem } from '../../../pure-view/SessionsSection';
import type { CustomAgent } from '../../../api/customAgents';
import { fetchCustomAgents, deleteCustomAgent } from '../../../api/customAgents';
import { CustomAgentCard } from './CustomAgentCard';
import { useState, useEffect } from 'react';
import { AgentEditor } from './AgentEditor';
import { PlusIcon } from '../../icons';
import { Pagination } from './Pagination';

export interface AgentPickerProps {
    agents: AgentDef[];
    loading: boolean;
    launchError: string;
    sessions: Record<string, AgentSessionInfo>;
    onLaunchHeadless: (agent: AgentDef) => void;
    onOpenSessions: (agentId: string) => void;
    onStopAgent: (agentId: string) => void;
    onConfigureAgent: (agentId: string) => void;
    onNavigateToView?: (view: string) => void;
    // External sessions from CLI/web opencode
    externalSessions?: ExternalOpencodeSession[];
    externalSessionsTotal?: number;
    externalSessionsPage?: number;
    // Sessions section props
    recentSessions?: SessionItem[];
    sessionsLoading?: boolean;
    onSelectSession?: (sessionId: string) => void;
    onNewSession?: () => void;
    onRefreshExternalSessions?: (page: number) => void;
}

export function AgentPicker({
    agents,
    loading,
    launchError,
    sessions,
    onLaunchHeadless,
    onOpenSessions,
    onStopAgent,
    onConfigureAgent,
    onNavigateToView,
    externalSessions = [],
    externalSessionsTotal = 0,
    externalSessionsPage = 1,
    recentSessions,
    sessionsLoading,
    onSelectSession,
    onNewSession,
    onRefreshExternalSessions,
}: AgentPickerProps) {
    const [customAgents, setCustomAgents] = useState<CustomAgent[]>([]);
    const [customAgentsLoading, setCustomAgentsLoading] = useState(true);
    const [showEditor, setShowEditor] = useState(false);
    const [editingAgent, setEditingAgent] = useState<CustomAgent | null>(null);

    useEffect(() => {
        loadCustomAgents();
    }, []);

    const loadCustomAgents = async () => {
        try {
            const agents = await fetchCustomAgents();
            setCustomAgents(agents);
        } catch (err) {
            console.error('Failed to load custom agents:', err);
        } finally {
            setCustomAgentsLoading(false);
        }
    };

    const handleCreateAgent = () => {
        setEditingAgent(null);
        setShowEditor(true);
    };

    const handleEditAgent = (agent: CustomAgent) => {
        setEditingAgent(agent);
        setShowEditor(true);
    };

    const handleDeleteAgent = async (agent: CustomAgent) => {
        if (!confirm(`Delete agent "${agent.name}"?`)) return;
        try {
            await deleteCustomAgent(agent.id);
            loadCustomAgents();
        } catch (err) {
            console.error('Failed to delete agent:', err);
            alert('Failed to delete agent');
        }
    };

    const handleSaveAgent = () => {
        setShowEditor(false);
        setEditingAgent(null);
        loadCustomAgents();
    };

    // Convert external sessions to SessionItem format for the SessionsSection
    // Show 5 sessions per page with pagination footer
    const PAGE_SIZE = 5;
    const allSessionItems: SessionItem[] = (recentSessions ?? externalSessions?.map(es => ({
        id: es.id,
        title: es.title || `Session ${es.slug || es.id.slice(0, 8)}`,
        preview: es.summary ? `${es.summary.files} files changed` : 'External session',
        createdAt: es.time?.created ? new Date(es.time.created * 1000).toLocaleDateString() : undefined,
    })) ?? []);
    const sessionItems = allSessionItems;
    const totalPages = Math.ceil((externalSessionsTotal || 0) / PAGE_SIZE);
    const currentPage = externalSessionsPage || 1;

    const handlePageChange = (newPage: number) => {
        if (onRefreshExternalSessions) {
            onRefreshExternalSessions(newPage);
        }
    };

    if (showEditor) {
        return (
            <div className="mcc-agent-view">
                <AgentEditor
                    agent={editingAgent}
                    onSave={handleSaveAgent}
                    onCancel={() => {
                        setShowEditor(false);
                        setEditingAgent(null);
                    }}
                />
            </div>
        );
    }

    return (
        <div className="mcc-agent-view">
            {/* Sessions Section - shown above agents */}
            {sessionItems.length > 0 && onSelectSession && (
                <>
                    <SessionsSection
                        sessions={sessionItems}
                        loading={sessionsLoading}
                        onSelectSession={onSelectSession}
                        onNewSession={onNewSession}
                        title="Sessions"
                    />
                    <Pagination
                        currentPage={currentPage}
                        totalPages={totalPages}
                        totalCount={externalSessionsTotal}
                        pageSize={PAGE_SIZE}
                        onPageChange={handlePageChange}
                        loading={sessionsLoading}
                    />
                </>
            )}

            {/* Custom Agents Section */}
            <div className="mcc-agent-header">
                <h2>Agents</h2>
                <button className="mcc-agent-add-btn" onClick={handleCreateAgent}>
                    <PlusIcon /> New Agent
                </button>
            </div>

            {customAgentsLoading ? (
                <div className="mcc-agent-loading">Loading agents...</div>
            ) : customAgents.length === 0 ? (
                <div className="mcc-agent-empty">
                    <p>No custom agents yet. Create one to get started!</p>
                    <button className="mcc-forward-btn mcc-agent-launch-btn" onClick={handleCreateAgent}>
                        Create Agent
                    </button>
                </div>
            ) : (
                <div className="mcc-agent-list">
                    {customAgents.map(agent => (
                        <CustomAgentCard
                            key={agent.id}
                            agent={agent}
                            onEdit={handleEditAgent}
                            onDelete={handleDeleteAgent}
                            onLaunch={onNavigateToView || (() => {})}
                        />
                    ))}
                </div>
            )}

            {/* Coding Tools Section (external agents) */}
            <div className="mcc-agent-header">
                <h2>Coding Tools</h2>
            </div>

            {loading && <div className="mcc-agent-loading">Loading agents...</div>}
            {launchError && <div className="mcc-agent-error">{launchError}</div>}

            <div className="mcc-agent-list">
                {agents.map(agent => {
                    const agentSession = sessions[agent.id];
                    return (
                        <div key={agent.id} className="mcc-agent-card">
                            <div className="mcc-agent-card-header">
                                <div className="mcc-agent-card-info">
                                    <span className="mcc-agent-card-name">{agent.name}</span>
                                    <span className={`mcc-agent-card-status ${agent.installed ? 'installed' : 'not-installed'}`}>
                                        {agent.installed ? 'Installed' : 'Not installed'}
                                    </span>
                                    {agentSession && (
                                        <span className={`mcc-agent-card-status ${agentSession.status}`}>
                                            {agentSession.status}
                                        </span>
                                    )}
                                </div>
                                <button
                                    className="mcc-agent-card-settings-icon"
                                    onClick={() => onConfigureAgent(agent.id)}
                                    title="Settings"
                                >
                                    <SettingsIcon />
                                </button>
                            </div>
                            <div className="mcc-agent-card-desc">{agent.description}</div>
                            <div className="mcc-agent-card-actions">
                                {agent.headless && agent.installed && agentSession && (
                                    <>
                                        <button
                                            className="mcc-forward-btn mcc-agent-launch-btn"
                                            onClick={() => onOpenSessions(agent.id)}
                                        >
                                            Open Sessions
                                        </button>
                                        <button
                                            className="mcc-agent-stop-btn"
                                            onClick={() => onStopAgent(agent.id)}
                                        >
                                            Stop
                                        </button>
                                    </>
                                )}
                                {/* Show Open Sessions for opencode when external sessions exist (even without internal session) */}
                                {agent.headless && agent.installed && !agentSession && agent.id === 'opencode' && externalSessions.length > 0 && (
                                    <button
                                        className="mcc-forward-btn mcc-agent-launch-btn"
                                        onClick={() => onOpenSessions(agent.id)}
                                    >
                                        Open Sessions ({externalSessions.length})
                                    </button>
                                )}
                                {agent.headless && agent.installed && !agentSession && !(agent.id === 'opencode' && externalSessions.length > 0) && (
                                    <button
                                        className="mcc-forward-btn mcc-agent-launch-btn"
                                        onClick={() => onLaunchHeadless(agent)}
                                    >
                                        Start Chat
                                    </button>
                                )}
                                {!agent.headless && agent.installed && (
                                    <span className="mcc-agent-card-note">Terminal-only agent</span>
                                )}
                                {!agent.installed && (
                                    <button
                                        className="mcc-forward-btn mcc-agent-configure-btn"
                                        onClick={() => onConfigureAgent(agent.id)}
                                    >
                                        Configure Path
                                    </button>
                                )}
                            </div>
                        </div>
                    );
                })}
            </div>
        </div>
    );
}
