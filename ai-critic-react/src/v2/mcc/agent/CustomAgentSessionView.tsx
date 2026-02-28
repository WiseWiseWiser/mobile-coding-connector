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
  const eventSourceRef = useRef<EventSource | null>(null);

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
    </div>
  );
}
