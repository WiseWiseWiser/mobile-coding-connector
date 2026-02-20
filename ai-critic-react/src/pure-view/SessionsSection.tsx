import './SessionsSection.css';

export interface SessionItem {
  id: string;
  title: string;
  preview: string;
  createdAt?: string;
}

export interface SessionsSectionProps {
  sessions: SessionItem[];
  loading?: boolean;
  onSelectSession: (sessionId: string) => void;
  onNewSession?: () => void;
  emptyMessage?: string;
  title?: string;
}

export function SessionsSection({
  sessions,
  loading = false,
  onSelectSession,
  onNewSession,
  emptyMessage = 'No sessions yet',
  title = 'Sessions',
}: SessionsSectionProps) {
  return (
    <div className="sessions-section">
      <div className="sessions-section__header">
        <h3 className="sessions-section__title">{title}</h3>
        {onNewSession && (
          <button
            className="sessions-section__new-btn"
            onClick={onNewSession}
            title="New session"
          >
            <PlusIcon />
          </button>
        )}
      </div>

      <div className="sessions-section__content">
        {loading ? (
          <div className="sessions-section__loading">
            <SpinnerIcon />
            <span>Loading sessions...</span>
          </div>
        ) : sessions.length === 0 ? (
          <div className="sessions-section__empty">{emptyMessage}</div>
        ) : (
          <ul className="sessions-section__list">
            {sessions.map((session) => (
              <li key={session.id} className="sessions-section__item">
                <button
                  className="sessions-section__card"
                  onClick={() => onSelectSession(session.id)}
                >
                  <div className="sessions-section__card-title">
                    {session.title}
                  </div>
                  {session.preview && (
                    <div className="sessions-section__card-preview">
                      {session.preview}
                    </div>
                  )}
                  {session.createdAt && (
                    <div className="sessions-section__card-meta">
                      {session.createdAt}
                    </div>
                  )}
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

// Icons
function PlusIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <line x1="12" y1="5" x2="12" y2="19" />
      <line x1="5" y1="12" x2="19" y2="12" />
    </svg>
  );
}

function SpinnerIcon() {
  return (
    <div className="sessions-section__spinner" />
  );
}
