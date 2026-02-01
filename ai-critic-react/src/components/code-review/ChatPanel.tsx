import { useState, useRef, useEffect } from 'react';
import { ModelSelector } from './ModelSelector';
import { ChatMessage } from './ChatMessage';
import type { ProviderInfo, ModelInfo } from './types';

interface Message {
    role: 'user' | 'assistant';
    content: string;
    thinking?: string; // AI reasoning/thinking content
}

interface ChatPanelProps {
    diffContext: string;
    provider: string;
    model: string;
    providers: ProviderInfo[];
    models: ModelInfo[];
    onProviderChange: (provider: string) => void;
    onModelChange: (model: string) => void;
    loading: boolean;
    hasFiles: boolean;
}

// Initial welcome message shown to user but not sent to API
const INITIAL_MESSAGE: Message = {
    role: 'assistant',
    content: `Hi! I'm your AI code review assistant. I can help you:

- **Understand** the code changes in your diff
- **Find issues** like bugs, security problems, or performance concerns
- **Suggest improvements** to make your code better
- **Explain** any part of the code you're curious about

Just ask me anything about your code changes!`
};

export function ChatPanel({ 
    diffContext, 
    provider, 
    model, 
    providers, 
    models, 
    onProviderChange, 
    onModelChange,
    loading: externalLoading,
    hasFiles,
}: ChatPanelProps) {
    const [messages, setMessages] = useState<Message[]>([INITIAL_MESSAGE]);
    const [input, setInput] = useState('');
    const [loading, setLoading] = useState(false);
    const messagesEndRef = useRef<HTMLDivElement>(null);
    const messagesContainerRef = useRef<HTMLDivElement>(null);
    const [shouldAutoScroll, setShouldAutoScroll] = useState(true);

    const scrollToBottom = () => {
        if (shouldAutoScroll) {
            messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
        }
    };

    // Check if user is near bottom of scroll
    const handleScroll = () => {
        const container = messagesContainerRef.current;
        if (container) {
            const { scrollTop, scrollHeight, clientHeight } = container;
            const isNearBottom = scrollHeight - scrollTop - clientHeight < 50;
            setShouldAutoScroll(isNearBottom);
        }
    };

    useEffect(() => {
        scrollToBottom();
    }, [messages, shouldAutoScroll]);

    const sendMessage = async (messageText: string) => {
        if (!messageText.trim() || loading) return;

        const userMessage = messageText.trim();
        const currentMessages = [...messages, { role: 'user' as const, content: userMessage }];
        setMessages(currentMessages);
        setLoading(true);
        setShouldAutoScroll(true);

        try {
            // Filter out the initial welcome message - it's only for UI display
            const messagesToSend = currentMessages.filter(msg => msg !== INITIAL_MESSAGE);
            
            const response = await fetch('/api/review/chat', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    messages: messagesToSend,
                    diffContext,
                    provider,
                    model,
                }),
            });

            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.error || 'Failed to get response');
            }

            // Handle streaming response
            const reader = response.body?.getReader();
            const decoder = new TextDecoder();
            let assistantMessage = '';
            let thinkingMessage = '';

            setMessages(prev => [...prev, { role: 'assistant', content: '', thinking: '' }]);

            if (reader) {
                while (true) {
                    const { done, value } = await reader.read();
                    if (done) break;

                    const chunk = decoder.decode(value, { stream: true });
                    // Parse SSE format
                    const lines = chunk.split('\n');
                    for (const line of lines) {
                        if (line.startsWith('data: ')) {
                            const data = line.slice(6);
                            if (data === '[DONE]') continue;
                            try {
                                const parsed = JSON.parse(data);
                                if (parsed.content) {
                                    if (parsed.type === 'thinking') {
                                        thinkingMessage += parsed.content;
                                    } else {
                                        assistantMessage += parsed.content;
                                    }
                                    setMessages(prev => {
                                        const newMessages = [...prev];
                                        newMessages[newMessages.length - 1] = {
                                            role: 'assistant',
                                            content: assistantMessage,
                                            thinking: thinkingMessage,
                                        };
                                        return newMessages;
                                    });
                                }
                            } catch {
                                // Ignore parse errors for incomplete chunks
                            }
                        }
                    }
                }
            }
        } catch (err) {
            setMessages(prev => [...prev, { 
                role: 'assistant', 
                content: `Error: ${err instanceof Error ? err.message : 'Unknown error'}` 
            }]);
        } finally {
            setLoading(false);
        }
    };

    const handleSend = async () => {
        if (!input.trim() || loading) return;
        const messageText = input.trim();
        setInput('');
        await sendMessage(messageText);
    };

    const handleReviewCode = async () => {
        await sendMessage('Review the code changes and point out any issues based on the configured rules.');
    };

    const handleKeyDown = (e: React.KeyboardEvent) => {
        // Send on Ctrl+Enter or Cmd+Enter
        if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
            e.preventDefault();
            handleSend();
        }
        // Regular Enter allows newline (default behavior)
    };

    const handleClear = () => {
        setMessages([INITIAL_MESSAGE]);
        setInput('');
    };

    const hasProviders = providers && providers.length > 0;
    const hasConversation = messages.length > 1; // More than just the initial message
    const isLoading = loading || externalLoading;

    return (
        <div style={{ 
            width: '400px',
            borderLeft: '1px solid #e5e5e5',
            display: 'flex',
            flexDirection: 'column',
            backgroundColor: '#fff',
        }}>
            {/* Header with model selector */}
            <div style={{
                padding: '12px 16px',
                borderBottom: '1px solid #e5e5e5',
            }}>
                <div style={{
                    fontWeight: 600,
                    fontSize: '14px',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '8px',
                    marginBottom: hasProviders ? '10px' : 0,
                }}>
                    <span>🤖</span>
                    <span>AI Assistant</span>
                </div>
                {hasProviders && (
                    <div style={{ display: 'flex', gap: '8px', alignItems: 'center', flexWrap: 'wrap' }}>
                        <ModelSelector
                            providers={providers}
                            models={models}
                            selectedProvider={provider}
                            selectedModel={model}
                            onProviderChange={onProviderChange}
                            onModelChange={onModelChange}
                        />
                        <button
                            onClick={handleReviewCode}
                            disabled={isLoading || !hasFiles}
                            style={{
                                padding: '6px 12px',
                                fontSize: '12px',
                                backgroundColor: '#2196F3',
                                color: 'white',
                                border: 'none',
                                borderRadius: '4px',
                                cursor: (isLoading || !hasFiles) ? 'not-allowed' : 'pointer',
                                opacity: (isLoading || !hasFiles) ? 0.7 : 1,
                            }}
                        >
                            {isLoading ? 'Analyzing...' : 'Review Code'}
                        </button>
                        {hasConversation && (
                            <button
                                onClick={handleClear}
                                disabled={isLoading}
                                style={{
                                    padding: '6px 12px',
                                    fontSize: '12px',
                                    backgroundColor: '#6b7280',
                                    color: 'white',
                                    border: 'none',
                                    borderRadius: '4px',
                                    cursor: isLoading ? 'not-allowed' : 'pointer',
                                    opacity: isLoading ? 0.7 : 1,
                                }}
                            >
                                Clear
                            </button>
                        )}
                    </div>
                )}
            </div>

            {/* Chat Messages */}
            <div 
                ref={messagesContainerRef}
                onScroll={handleScroll}
                style={{
                    flex: 1,
                    overflow: 'auto',
                    padding: '12px',
                    display: 'flex',
                    flexDirection: 'column',
                    gap: '12px',
                }}
            >
                {messages.length === 0 && (
                    <div style={{
                        color: '#9ca3af',
                        fontSize: '13px',
                        textAlign: 'center',
                        padding: '20px',
                    }}>
                        Ask questions about the code changes...
                    </div>
                )}
                {messages.map((msg, idx) => (
                    <ChatMessage key={idx} role={msg.role} content={msg.content} thinking={msg.thinking} />
                ))}
                {loading && (
                    <div style={{
                        color: '#6b7280',
                        fontSize: '12px',
                        padding: '8px',
                    }}>
                        Thinking...
                    </div>
                )}
                <div ref={messagesEndRef} />
            </div>

            {/* Input */}
            <div style={{
                padding: '12px',
                borderTop: '1px solid #e5e5e5',
            }}>
                <div style={{
                    display: 'flex',
                    gap: '8px',
                }}>
                    <textarea
                        value={input}
                        onChange={(e) => setInput(e.target.value)}
                        onKeyDown={handleKeyDown}
                        placeholder="Ask about the code... (Ctrl+Enter to send)"
                        style={{
                            flex: 1,
                            padding: '8px 12px',
                            fontSize: '13px',
                            border: '1px solid #d1d5db',
                            borderRadius: '6px',
                            resize: 'none',
                            minHeight: '40px',
                            maxHeight: '100px',
                            fontFamily: 'inherit',
                        }}
                        rows={1}
                    />
                    <button
                        onClick={handleSend}
                        disabled={loading || !input.trim()}
                        style={{
                            padding: '8px 16px',
                            fontSize: '13px',
                            backgroundColor: '#2196F3',
                            color: 'white',
                            border: 'none',
                            borderRadius: '6px',
                            cursor: (loading || !input.trim()) ? 'not-allowed' : 'pointer',
                            opacity: (loading || !input.trim()) ? 0.7 : 1,
                        }}
                    >
                        Send
                    </button>
                </div>
            </div>
        </div>
    );
}
