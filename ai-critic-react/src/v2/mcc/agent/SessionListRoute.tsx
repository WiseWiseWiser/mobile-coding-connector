import { useOutletContext, useParams } from 'react-router-dom';
import { useState, useEffect } from 'react';
import type { AgentOutletContext } from './AgentLayout';
import { SessionList } from './SessionList';
import { AgentPicker } from './AgentPicker';
import { ExternalSessionList } from './ExternalSessionList';
import { fetchAgentSessions, launchCustomAgent, type CustomAgentSession } from '../../../api/customAgents';
import { useProjectDir } from '../../../hooks/project/useProjectDir';
import { ActionButton } from '../../../pure-view/buttons/ActionButton';

export function SessionListRoute() {
    const ctx = useOutletContext<AgentOutletContext>();
    const { projectDir } = useProjectDir();
    const params = useParams<{ agentId?: string }>();
    const agentId = params.agentId || '';

    const session = ctx.sessions[agentId];
    const isCustomAgent = !ctx.agents.some(a => a.id === agentId);

    const [customSessions, setCustomSessions] = useState<CustomAgentSession[]>([]);
    const [loadingCustom, setLoadingCustom] = useState(false);
    const [launching, setLaunching] = useState(false);

    useEffect(() => {
        if (!isCustomAgent || !agentId) return;
        setLoadingCustom(true);
        fetchAgentSessions(agentId)
            .then(setCustomSessions)
            .catch(() => setCustomSessions([]))
            .finally(() => setLoadingCustom(false));
    }, [isCustomAgent, agentId]);

    const handleNewSession = async () => {
        if (!projectDir) {
            alert('Please select a project first');
            return;
        }
        setLaunching(true);
        try {
            const result = await launchCustomAgent(agentId, projectDir);
            ctx.navigateToView(`${agentId}/${result.sessionId}`);
        } catch (err) {
            alert('Failed to launch: ' + (err instanceof Error ? err.message : String(err)));
        } finally {
            setLaunching(false);
        }
    };

    // Custom agent session list
    if (isCustomAgent) {
        if (loadingCustom) {
            return <div className="mcc-agent-view"><div className="mcc-agent-loading">Loading sessions...</div></div>;
        }
        return (
            <div className="mcc-agent-view">
                <div className="mcc-section-header">
                    <button className="mcc-back-btn" onClick={() => ctx.navigateToView('')}>
                        &larr;
                    </button>
                    <h2>{agentId}</h2>
                </div>
                {customSessions.length === 0 ? (
                    <div className="mcc-agent-empty" style={{ padding: '0 16px' }}>
                        <p>No sessions yet for this agent.</p>
                    </div>
                ) : (
                    <div className="mcc-agent-list">
                        {customSessions.map(s => (
                            <div
                                key={s.id}
                                className="mcc-agent-card"
                                style={{ cursor: 'pointer' }}
                                onClick={() => ctx.navigateToView(`${agentId}/${s.id}`)}
                            >
                                <div className="mcc-agent-card-header">
                                    <div className="mcc-agent-card-info">
                                        <span className="mcc-agent-card-name">{s.id.slice(0, 12)}</span>
                                        <span className={`mcc-agent-card-status ${s.status === 'running' ? 'installed' : ''}`}>
                                            {s.status}
                                        </span>
                                    </div>
                                </div>
                                <div className="mcc-agent-card-desc">
                                    {s.project_dir} &middot; {new Date(s.created_at).toLocaleString()}
                                </div>
                            </div>
                        ))}
                    </div>
                )}
                <div style={{ padding: '12px 16px' }}>
                    <ActionButton onClick={handleNewSession} disabled={launching}>
                        {launching ? 'Starting...' : 'New Session'}
                    </ActionButton>
                    {projectDir && (
                        <div style={{ marginTop: 8, fontSize: 12, color: '#888', fontFamily: 'monospace', wordBreak: 'break-all' }}>
                            {projectDir}
                        </div>
                    )}
                </div>
            </div>
        );
    }

    // For opencode with external sessions but no internal session, show external session list
    const hasExternalSessions = agentId === 'opencode' && ctx.externalSessions.length > 0;
    
    if (!session && hasExternalSessions) {
        const agent = ctx.agents.find(a => a.id === agentId);
        return (
            <ExternalSessionList
                projectName={ctx.projectName}
                onBack={() => ctx.navigateToView('')}
                onSelectSession={(sessionId) => {
                    ctx.navigateToView(`${agentId}/${sessionId}`);
                }}
                onNewSession={() => {
                    if (agent) {
                        ctx.onLaunchHeadless(agent);
                    }
                }}
            />
        );
    }

    // If no session for this agent and no external sessions, fall back to agent picker
    if (!session) {
        return (
            <AgentPicker
                agents={ctx.agents}
                loading={ctx.agentsLoading}
                launchError={ctx.launchError}
                sessions={ctx.sessions}
                onLaunchHeadless={ctx.onLaunchHeadless}
                onOpenSessions={(aid) => ctx.navigateToView(aid)}
                onStopAgent={ctx.onStopAgent}
                onConfigureAgent={(aid) => ctx.navigateToView(`${aid}/settings`)}
                externalSessions={ctx.externalSessions}
            />
        );
    }

    return (
        <SessionList
            session={session}
            projectName={ctx.projectName}
            onBack={() => ctx.navigateToView('')}
            onStop={() => ctx.onStopAgent(agentId)}
            onSelectSession={(sid) => ctx.navigateToView(`${agentId}/${sid}`)}
            onSessionUpdate={(updated) => ctx.setSession(agentId, updated)}
            onSettings={() => ctx.navigateToView(`${agentId}/settings`)}
        />
    );
}
