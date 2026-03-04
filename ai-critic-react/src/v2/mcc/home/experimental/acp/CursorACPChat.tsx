import { useRef, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { ACPChat, type ACPChatHandle } from './chat/ACPChat';

export function CursorACPChat() {
    const { projectName, sessionId } = useParams<{ projectName?: string; sessionId?: string }>();
    const chatRef = useRef<ACPChatHandle>(null);
    const connectFired = useRef(false);

    const isNewSession = sessionId === 'new';

    useEffect(() => {
        if (!isNewSession) return;
        if (connectFired.current) return;
        connectFired.current = true;

        chatRef.current?.connect(undefined, projectName);
    }, [isNewSession, projectName]);

    return (
        <ACPChat
            ref={chatRef}
            title="Cursor UI (ACP)"
            agentName="Cursor"
            apiPrefix="/api/agent/acp/cursor"
        />
    );
}
