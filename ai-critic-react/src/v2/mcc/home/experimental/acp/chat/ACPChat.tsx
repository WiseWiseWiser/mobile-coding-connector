import { useImperativeHandle, forwardRef } from 'react';
import { ACPChatHeader } from './ACPChatHeader';
import { ACPChatToolbar } from './ACPChatToolbar';
import { ACPChatConnectLogs } from './ACPChatConnectLogs';
import { ACPChatMessages } from './ACPChatMessages';
import { ACPChatInput } from './ACPChatInput';
import { useACPChat } from './useACPChat';
import '../ACPUI.css';

export type { ChatMessage } from './ACPChatTypes';

export interface ACPChatHandle {
    connect(resumeSessionId?: string, projectName?: string, worktreeId?: string): void;
}

export interface ACPChatProps {
    title: string;
    agentName: string;
    apiPrefix: string;
    emptyConnectedMessage?: string;
}

export const ACPChat = forwardRef<ACPChatHandle, ACPChatProps>(function ACPChat({
    title,
    agentName,
    apiPrefix,
    emptyConnectedMessage = `Send a message to start coding with ${agentName} agent.`,
}, ref) {
    const chat = useACPChat({ agentName, apiPrefix });

    useImperativeHandle(ref, () => ({
        connect(resumeSessionId?: string, projectName?: string, worktreeId?: string) {
            chat.connect(resumeSessionId, projectName, worktreeId);
        },
    }));

    return (
        <div className="acp-ui-container">
            <ACPChatHeader
                title={title}
                status={chat.status}
                statusMessage={chat.statusMessage}
                sessionId={chat.sessionId}
                isNewSession={chat.isNewSession}
                onBack={chat.handleBack}
            />

            <ACPChatToolbar
                models={chat.models}
                selectedModel={chat.selectedModel}
                onModelSelect={chat.handleModelSelect}
                modelPlaceholder={chat.model || 'Select model...'}
                yoloMode={chat.yoloMode}
                onYoloToggle={chat.handleYoloToggle}
                debugMode={chat.debugMode}
                onDebugToggle={() => chat.setDebugMode(!chat.debugMode)}
                dir={chat.dir}
                status={chat.status}
            />

            {chat.loadErrors.length > 0 && (
                <div style={{ padding: '6px 12px', background: 'rgba(248,113,113,0.1)', borderRadius: 6, margin: '0 12px' }}>
                    {chat.loadErrors.map((err, i) => (
                        <div key={i} style={{ fontSize: 12, color: 'var(--mcc-accent-red, #f87171)' }}>{err}</div>
                    ))}
                </div>
            )}

            <ACPChatConnectLogs
                connectLogs={chat.connectLogs}
                debugLogs={chat.debugLogs}
                debugMode={chat.debugMode}
                showConnectLogs={chat.showConnectLogs}
                onDismiss={() => chat.setShowConnectLogs(false)}
            />

            <ACPChatMessages
                messages={chat.messages}
                status={chat.status}
                agentName={agentName}
                emptyConnectedMessage={emptyConnectedMessage}
                chatContainerRef={chat.chatContainerRef}
                onScroll={chat.handleChatScroll}
            />

            <ACPChatInput
                input={chat.input}
                onInputChange={chat.setInput}
                onSend={chat.sendPrompt}
                onCancel={chat.cancelPrompt}
                isProcessing={chat.isProcessing}
                status={chat.status}
            />
        </div>
    );
});
