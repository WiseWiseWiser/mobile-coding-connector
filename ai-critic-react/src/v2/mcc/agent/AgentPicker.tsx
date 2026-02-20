import type { AgentDef, AgentSessionInfo, ExternalOpencodeSession } from '../../../api/agents';
import { SettingsIcon } from '../../icons';
import { SessionsSection, type SessionItem } from '../../../pure-view/SessionsSection';

export interface AgentPickerProps {
    agents: AgentDef[];
    loading: boolean;
    launchError: string;
    sessions: Record<string, AgentSessionInfo>;
    onLaunchHeadless: (agent: AgentDef) => void;
    onOpenSessions: (agentId: string) => void;
    onStopAgent: (agentId: string) => void;
    onConfigureAgent: (agentId: string) => void;
    // External sessions from CLI/web opencode
    externalSessions?: ExternalOpencodeSession[];
    // Sessions section props
    recentSessions?: SessionItem[];
    sessionsLoading?: boolean;
    onSelectSession?: (sessionId: string) => void;
    onNewSession?: () => void;
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
    externalSessions = [],
    recentSessions,
    sessionsLoading,
    onSelectSession,
    onNewSession,
}: AgentPickerProps) {
    // Convert external sessions to SessionItem format for the SessionsSection
    const sessionItems: SessionItem[] = recentSessions ?? externalSessions?.map(es => ({
        id: es.id,
        title: es.title || `Session ${es.slug || es.id.slice(0, 8)}`,
        preview: es.summary ? `${es.summary.files} files changed` : 'External session',
        createdAt: es.time?.created ? new Date(es.time.created * 1000).toLocaleDateString() : undefined,
    })) ?? [];

    return (
        <div className="mcc-agent-view">
            {/* Sessions Section - shown above agents */}
            {sessionItems.length > 0 && onSelectSession && (
                <SessionsSection
                    sessions={sessionItems}
                    loading={sessionsLoading}
                    onSelectSession={onSelectSession}
                    onNewSession={onNewSession}
                    title="Sessions"
                />
            )}

            <div className="mcc-agent-header">
                <h2>Agents</h2>
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
