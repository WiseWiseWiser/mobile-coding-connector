import { useOutletContext, useParams } from 'react-router-dom';
import { useState, useEffect } from 'react';
import type { AgentOutletContext } from './AgentLayout';
import { AgentChat } from './AgentChat';
import { AgentPicker } from './AgentPicker';
import { fetchOpencodeServer } from '../../../api/agents';

export function AgentChatRoute() {
    const ctx = useOutletContext<AgentOutletContext>();
    const params = useParams<{ agentId?: string; sessionId?: string }>();
    const agentId = params.agentId || '';
    const sessionId = params.sessionId || '';

    const session = ctx.sessions[agentId];

    // For external sessions, we need to get the opencode server port
    const [externalServerPort, setExternalServerPort] = useState<number | null>(null);
    const [loadingExternal, setLoadingExternal] = useState(false);

    const isExternalSession = !session && agentId === 'opencode' && ctx.externalSessions.length > 0;

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

    // If no session for this agent, fall back to agent picker
    // But for opencode with external sessions, we show the chat if server is ready
    if (!session) {
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
