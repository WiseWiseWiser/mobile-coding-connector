import type { AgentDef, AgentSessionInfo } from '../../../api/agents';
import { SettingsIcon } from '../../icons';

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
    externalSessions?: { id: string }[];
}

export function AgentPicker({ agents, loading, launchError, sessions, onLaunchHeadless, onOpenSessions, onStopAgent, onConfigureAgent, externalSessions = [] }: AgentPickerProps) {
    return (
        <div className="mcc-agent-view">
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
