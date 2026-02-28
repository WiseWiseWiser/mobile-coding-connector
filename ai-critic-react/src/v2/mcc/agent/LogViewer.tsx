import { useRef, useEffect } from 'react';

interface LogViewerProps {
  logs: string[];
  loading?: boolean;
  maxHeight?: string;
}

export function LogViewer({ logs, loading, maxHeight = '400px' }: LogViewerProps) {
  const scrollRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom when new logs arrive
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [logs]);

  const formatLogLine = (line: string): React.ReactElement => {
    // Colorize log levels
    let className = 'mcc-log-line';
    if (line.includes(' ERROR ') || line.includes(' error ')) {
      className += ' mcc-log-error';
    } else if (line.includes(' WARN ') || line.includes(' warn ')) {
      className += ' mcc-log-warn';
    } else if (line.includes(' INFO ') || line.includes(' info ')) {
      className += ' mcc-log-info';
    }

    return <div className={className}>{line}</div>;
  };

  return (
    <div className="mcc-log-viewer">
      <div 
        ref={scrollRef}
        className="mcc-log-content"
        style={{ maxHeight }}
      >
        {logs.length === 0 ? (
          <div className="mcc-log-empty">
            {loading ? 'Waiting for logs...' : 'No logs yet'}
          </div>
        ) : (
          logs.map((line, index) => (
            <div key={index}>{formatLogLine(line)}</div>
          ))
        )}
      </div>
      {loading && logs.length > 0 && (
        <div className="mcc-log-loading">
          <span className="mcc-log-spinner" /> Waiting for more logs...
        </div>
      )}
    </div>
  );
}
