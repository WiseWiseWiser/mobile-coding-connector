import { useRef, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { useCurrent } from '../../../../../hooks/useCurrent';
import { useV2Context } from '../../../../V2Context';
import { ACPChat, type ACPChatHandle } from './ACPChat';

export function CursorACPChat() {
    const { projectName, sessionId } = useParams<{ projectName?: string; sessionId?: string }>();
    const { getProjectDir } = useV2Context();
    const getProjectDirRef = useCurrent(getProjectDir);
    const chatRef = useRef<ACPChatHandle>(null);
    const connectFired = useRef(false);

    const isNewSession = sessionId === 'new';

    console.log("DEBUG CursorACPChat render", { projectName, sessionId, isNewSession });

    useEffect(() => {
        if (!isNewSession) return;
        if (connectFired.current) return;
        connectFired.current = true;

        (async () => {
            const cwd = projectName ? await getProjectDirRef.current(projectName) : '';
            console.log("DEBUG CursorACPChat firing connect, cwd=", cwd);
            chatRef.current?.connect(undefined, cwd);
        })();
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
