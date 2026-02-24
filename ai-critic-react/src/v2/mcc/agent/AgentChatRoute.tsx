import { useOutletContext, useParams } from 'react-router-dom';
import { useState, useEffect } from 'react';
import type { AgentOutletContext } from './AgentLayout';
import { AgentChat } from './AgentChat';
import { AgentPicker } from './AgentPicker';
import { fetchOpencodeServer } from '../../../api/agents';
import { fetchCustomAgentSessions, type CustomAgentSession } from '../../../api/customAgents';

export function AgentChatRoute() {
    const ctx = useOutletContext<AgentOutletContext>();
    const params = useParams<{ agentId?: string; sessionId?: string }>();
    const agentId = params.agentId || '';
    const sessionId = params.sessionId || '';

    const session = ctx.sessions[agentId];

    // For external sessions, we need to get the opencode server port
    const [externalServerPort, setExternalServerPort] = useState<number | null>(null);
    const [loadingExternal, setLoadingExternal] = useState(false);

    // For custom agent sessions
    const [customSessions, setCustomSessions] = useState<CustomAgentSession[]>([]);
    const [loadingCustom, setLoadingCustom] = useState(false);

    const isExternalSession = !session && agentId === 'opencode' && ctx.externalSessions.length > 0;

    // Check if this is a custom agent
    const isCustomAgent = agentId.startsWith('custom-agent-') || (agentId === 'go-api-refactorer');

    useEffect(() => {
        if (isExternalSession && !externalServerPort) {
            setLoadingExternal(true);
            fetchOpencodeServer()
                .then(info => {
                    if (info && info.port) {
                        setExternalServerPort(info.port);
                    }
                    setLoadingExternal(false);
                })
                .catch(() => setLoadingExternal(false));
        }
    }, [isExternalSession, externalServerPort]);

    // Fetch custom agent sessions when needed
    useEffect(() => {
        if (isCustomAgent && customSessions.length === 0) {
            setLoadingCustom(true);
            fetchCustomAgentSessions()
                .then(sessions => {
                    setCustomSessions(sessions);
                    setLoadingCustom(false);
                })
                .catch(() => setLoadingCustom(false));
        }
    }, [isCustomAgent, customSessions.length]);

    // Find matching custom session for this agent
    const customSession = customSessions.find(s => 
        s.agent_id === agentId || 
        s.id.includes(agentId) ||
        agentId.includes(s.agent_id)
    );

    // If no session for this agent, fall back to agent picker
    // But for opencode with external sessions, we show the chat if server is ready
    if (!session) {
        // Handle custom agent
        if (isCustomAgent) {
            if (loadingCustom) {
                return (
                    <div className="mcc-agent-view">
                        <div className="mcc-loading">Loading...</div>
                    </div>
                );
            }
            if (!customSession) {
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
                        onNavigateToView={ctx.navigateToView}
                        externalSessions={ctx.externalSessions}
                    />
                );
            }
            return (
                <AgentChat
                    session={{
                        id: customSession.id,
                        agent_name: customSession.agent_name,
                        status: customSession.status,
                        port: customSession.port,
                    } as any}
                    projectName={ctx.projectName}
                    opencodeSID={sessionId}
                    onStop={() => {}}
                    onBack={() => ctx.navigateToView('')}
                    onSessionUpdate={() => {}}
                />
            );
        }

        if (isExternalSession) {
            if (!externalServerPort) {
                return (
                    <AgentChat
                        session={{
                            id: 'external',
                            agent_name: 'opencode',
                            status: 'running',
                            port: 0,
                        } as any}
                        projectName={ctx.projectName}
                        opencodeSID={sessionId}
                        onStop={() => {}}
                        onBack={() => ctx.navigateToView(agentId)}
                        onSessionUpdate={() => {}}
                        connecting={loadingExternal}
                    />
                );
            }
            // For external session, create a mock session with the server port
            return (
                <AgentChat
                    session={{
                        id: 'external',
                        agent_name: 'opencode',
                        status: 'running',
                        port: externalServerPort,
                    } as any}
                    projectName={ctx.projectName}
                    opencodeSID={sessionId}
                    onStop={() => {}}
                    onBack={() => ctx.navigateToView(agentId)}
                    onSessionUpdate={() => {}}
                />
            );
        }

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
                onNavigateToView={ctx.navigateToView}
                externalSessions={ctx.externalSessions}
            />
        );
    }

    return (
        <AgentChat
            session={session}
            projectName={ctx.projectName}
            opencodeSID={sessionId}
            onStop={() => ctx.onStopAgent(agentId)}
            onBack={() => ctx.navigateToView(agentId)}
            onSessionUpdate={(updated) => ctx.setSession(agentId, updated)}
        />
    );
}
