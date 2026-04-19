import { useState, useEffect, useRef } from 'react';
import type { CustomAgentSession } from '../../../api/customAgents';
import { deleteCustomAgentSession } from '../../../api/customAgents';
import { AgentChatHeader } from './AgentChatHeader';
import { LogViewer } from './LogViewer';

interface CustomAgentSessionViewProps {
  session: CustomAgentSession;
  projectName: string | null;
  onBack: () => void;
}

export function CustomAgentSessionView({ session, projectName, onBack }: CustomAgentSessionViewProps) {
  const [logs, setLogs] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [messageInput, setMessageInput] = useState('');
  const [sending, setSending] = useState(false);
  const eventSourceRef = useRef<EventSource | null>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);

  // Connect to SSE for real-time logs
  useEffect(() => {
    if (session.status !== 'running' && session.status !== 'starting') {
      return;
    }

    // Create EventSource for logs
    const url = `/api/custom-agents/sessions/${session.id}/logs`;
    const es = new EventSource(url);
    eventSourceRef.current = es;

    es.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.type === 'log') {
          setLogs(prev => [...prev, data.message]);
        } else if (data.type === 'status') {
          // Handle status updates
        }
      } catch {
        // If not JSON, treat as plain log line
        setLogs(prev => [...prev, event.data]);
      }
    };

    es.onerror = () => {
      // Connection error, will auto-retry
    };

    return () => {
      es.close();
      eventSourceRef.current = null;
    };
  }, [session.id, session.status]);

  const handleStop = async () => {
    setLoading(true);
    setError(null);
    try {
      await deleteCustomAgentSession(session.id);
      onBack();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to stop session');
    } finally {
      setLoading(false);
    }
  };

  const handleSendMessage = async () => {
    if (!messageInput.trim() || sending) return;

    setSending(true);
    setError(null);

    try {
      const response = await fetch(`/api/custom-agents/sessions/${encodeURIComponent(session.id)}/message`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ message: messageInput.trim() }),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `Failed to send message: ${response.status}`);
      }

      // Clear input after successful send
      setMessageInput('');
      // Focus back on input
      inputRef.current?.focus();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to send message');
    } finally {
      setSending(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSendMessage();
    }
  };

  const getStatusClass = (status: string) => {
    switch (status) {
      case 'running': return 'mcc-agent-status-running';
      case 'starting': return 'mcc-agent-status-starting';
      case 'stopped': return 'mcc-agent-status-stopped';
      case 'error': return 'mcc-agent-status-error';
      default: return '';
    }
  };

  return (
    <div className="mcc-agent-session-view">
      <AgentChatHeader
        agentName={session.agent_name}
        projectName={projectName}
        onBack={onBack}
        onStop={session.status === 'running' || session.status === 'starting' ? handleStop : undefined}
        stopLabel={loading ? 'Stopping...' : 'Stop'}
      />

      {error && (
        <div className="mcc-agent-error-banner">
          {error}
        </div>
      )}

      <div className="mcc-agent-session-info">
        <div className="mcc-agent-session-meta">
          <span className={`mcc-agent-session-status ${getStatusClass(session.status)}`}>
            {session.status}
          </span>
          <span className="mcc-agent-session-id">ID: {session.id.slice(0, 8)}</span>
          <span className="mcc-agent-session-port">Port: {session.port}</span>
        </div>
        <div className="mcc-agent-session-project">
          Project: {session.project_dir}
        </div>
        <div className="mcc-agent-session-created">
          Created: {new Date(session.created_at).toLocaleString()}
        </div>
      </div>

      <div className="mcc-agent-logs-section">
        <h3>Session Logs</h3>
        <LogViewer logs={logs} loading={session.status === 'starting' && logs.length === 0} />
      </div>

      {/* Message Input Area */}
      {session.status === 'running' && (
        <div className="mcc-agent-input-area">
          <textarea
            ref={inputRef}
            className="mcc-agent-input"
            placeholder="Type a message to the agent... (Enter to send, Shift+Enter for new line)"
            value={messageInput}
            onChange={(e) => setMessageInput(e.target.value)}
            onKeyDown={handleKeyDown}
            disabled={sending}
            rows={1}
          />
          <button
            className="mcc-agent-send-btn"
            onClick={handleSendMessage}
            disabled={!messageInput.trim() || sending}
          >
            {sending ? 'Sending...' : 'Send'}
          </button>
        </div>
      )}
    </div>
  );
}
