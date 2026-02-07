import type { AgentDef, AgentSessionInfo } from '../../../api/agents';
import { FolderIcon } from '../../icons';

export interface AgentPickerProps {
    agents: AgentDef[];
    loading: boolean;
    projectName: string | null;
    launchError: string;
    activeSession: AgentSessionInfo | null;
    onLaunchHeadless: (agent: AgentDef) => void;
    onResumeChat: () => void;
    onStopSession: () => void;
}

export function AgentPicker({ agents, loading, projectName, launchError, activeSession, onLaunchHeadless, onResumeChat, onStopSession }: AgentPickerProps) {
    return (
        <div className="mcc-agent-view">
            <div className="mcc-agent-header">
                <h2>Agents</h2>
                <div className="mcc-agent-project-badge">
                    <FolderIcon />
                    <span>{projectName}</span>
                </div>
            </div>

            {loading && <div className="mcc-agent-loading">Loading agents...</div>}
            {launchError && <div className="mcc-agent-error">{launchError}</div>}

            {/* Active session banner */}
            {activeSession && (
                <div className="mcc-agent-active-session">
                    <div className="mcc-agent-active-session-info">
                        <span className="mcc-agent-active-session-label">Active session</span>
                        <span className={`mcc-agent-active-session-status ${activeSession.status}`}>
                            {activeSession.status}
                        </span>
                    </div>
                    <div className="mcc-agent-active-session-actions">
                        <button className="mcc-forward-btn" onClick={onResumeChat}>
                            Resume Chat
                        </button>
                        <button className="mcc-agent-stop-btn" onClick={onStopSession}>
                            Stop
                        </button>
                    </div>
                </div>
            )}

            <div className="mcc-agent-list">
                {agents.map(agent => (
                    <div key={agent.id} className="mcc-agent-card">
                        <div className="mcc-agent-card-header">
                            <div className="mcc-agent-card-info">
                                <span className="mcc-agent-card-name">{agent.name}</span>
                                <span className={`mcc-agent-card-status ${agent.installed ? 'installed' : 'not-installed'}`}>
                                    {agent.installed ? 'Installed' : 'Not installed'}
                                </span>
                            </div>
                        </div>
                        <div className="mcc-agent-card-desc">{agent.description}</div>
                        <div className="mcc-agent-card-actions">
                            {agent.headless && agent.installed && !activeSession && (
                                <button
                                    className="mcc-forward-btn mcc-agent-launch-btn"
                                    onClick={() => onLaunchHeadless(agent)}
                                >
                                    Start Chat
                                </button>
                            )}
                            {agent.headless && agent.installed && activeSession && (
                                <button
                                    className="mcc-forward-btn mcc-agent-launch-btn"
                                    onClick={onResumeChat}
                                >
                                    Open Sessions
                                </button>
                            )}
                            {!agent.headless && agent.installed && (
                                <span className="mcc-agent-card-note">Terminal-only agent</span>
                            )}
                            {!agent.installed && (
                                <span className="mcc-agent-card-note">Not available</span>
                            )}
                        </div>
                    </div>
                ))}
            </div>
        </div>
    );
}
