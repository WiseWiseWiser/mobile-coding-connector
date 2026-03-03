import { useRef, useEffect } from 'react';
import type { ConnectionStatus } from './ACPChatTypes';

export interface ACPChatInputProps {
    input: string;
    onInputChange: (value: string) => void;
    onSend: () => void;
    onCancel: () => void;
    isProcessing: boolean;
    status: ConnectionStatus;
}

export function ACPChatInput({ input, onInputChange, onSend, onCancel, isProcessing, status }: ACPChatInputProps) {
    const inputRef = useRef<HTMLTextAreaElement>(null);

    useEffect(() => {
        if (status === 'connected' && !isProcessing) {
            inputRef.current?.focus();
        }
    }, [status, isProcessing]);

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            onSend();
        }
    };

    return (
        <div className="acp-ui-input-area">
            <textarea
                ref={inputRef}
                className="acp-ui-input"
                placeholder={status === 'connected' ? 'Type a message... (Enter to send, Shift+Enter for newline)' : 'Connecting...'}
                value={input}
                onChange={e => onInputChange(e.target.value)}
                onKeyDown={handleKeyDown}
                disabled={status !== 'connected' || isProcessing}
                rows={3}
            />
            <div className="acp-ui-input-actions">
                {isProcessing ? (
                    <button className="mcc-btn-secondary" onClick={onCancel}>Cancel</button>
                ) : (
                    <button
                        className="mcc-btn-primary"
                        onClick={onSend}
                        disabled={!input.trim() || status !== 'connected'}
                    >
                        Send
                    </button>
                )}
            </div>
        </div>
    );
}
