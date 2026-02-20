import { useState } from 'react';
import { SessionsSection, type SessionItem } from '../pure-view/SessionsSection';
import './SessionsSectionMockup.css';

// Mock session data
const mockSessions: SessionItem[] = [
  {
    id: 'sess-001',
    title: 'Session 5',
    preview: 'Fix the login authentication bug in the user service...',
    createdAt: '2 hours ago',
  },
  {
    id: 'sess-002',
    title: 'Session 4',
    preview: 'Add dark mode support to the dashboard components...',
    createdAt: '5 hours ago',
  },
  {
    id: 'sess-003',
    title: 'Session 3',
    preview: 'Refactor the API client to use async/await...',
    createdAt: '1 day ago',
  },
  {
    id: 'sess-004',
    title: 'Session 2',
    preview: 'Set up the initial project structure and dependencies...',
    createdAt: '2 days ago',
  },
  {
    id: 'sess-005',
    title: 'Session 1',
    preview: 'Initial conversation about project requirements...',
    createdAt: '3 days ago',
  },
];

// Empty mock sessions for testing empty state
const emptySessions: SessionItem[] = [];

export function SessionsSectionMockup() {
  const [selectedSession, setSelectedSession] = useState<string | null>(null);
  const [showNewSessionModal, setShowNewSessionModal] = useState(false);

  const handleSelectSession = (sessionId: string) => {
    setSelectedSession(sessionId);
    console.log('Selected session:', sessionId);
  };

  const handleNewSession = () => {
    setShowNewSessionModal(true);
    console.log('Creating new session...');
  };

  return (
    <div className="sessions-section-mockup">
      <div className="sessions-section-mockup__header">
        <h1>Sessions Section Component</h1>
        <p>A reusable pure-view component for displaying session lists</p>
      </div>

      <div className="sessions-section-mockup__examples">
        {/* Example 1: With Sessions */}
        <div className="sessions-section-mockup__example">
          <div className="sessions-section-mockup__example-header">
            <h3>With Sessions</h3>
            <span className="sessions-section-mockup__badge">Default</span>
          </div>
          <div className="sessions-section-mockup__example-content">
            <SessionsSection
              sessions={mockSessions}
              onSelectSession={handleSelectSession}
              onNewSession={handleNewSession}
            />
          </div>
          {selectedSession && (
            <div className="sessions-section-mockup__selection">
              Selected: <code>{selectedSession}</code>
            </div>
          )}
        </div>

        {/* Example 2: Empty State */}
        <div className="sessions-section-mockup__example">
          <div className="sessions-section-mockup__example-header">
            <h3>Empty State</h3>
            <span className="sessions-section-mockup__badge">No Sessions</span>
          </div>
          <div className="sessions-section-mockup__example-content">
            <SessionsSection
              sessions={emptySessions}
              onSelectSession={handleSelectSession}
              onNewSession={handleNewSession}
            />
          </div>
        </div>

        {/* Example 3: Loading State */}
        <div className="sessions-section-mockup__example">
          <div className="sessions-section-mockup__example-header">
            <h3>Loading State</h3>
            <span className="sessions-section-mockup__badge">Loading</span>
          </div>
          <div className="sessions-section-mockup__example-content">
            <SessionsSection
              sessions={emptySessions}
              loading={true}
              onSelectSession={handleSelectSession}
              onNewSession={handleNewSession}
            />
          </div>
        </div>

        {/* Example 4: Without New Session Button */}
        <div className="sessions-section-mockup__example">
          <div className="sessions-section-mockup__example-header">
            <h3>Read-only</h3>
            <span className="sessions-section-mockup__badge">No Action</span>
          </div>
          <div className="sessions-section-mockup__example-content">
            <SessionsSection
              sessions={mockSessions.slice(0, 3)}
              onSelectSession={handleSelectSession}
            />
          </div>
        </div>
      </div>

      {showNewSessionModal && (
        <div
          className="sessions-section-mockup__modal-overlay"
          onClick={() => setShowNewSessionModal(false)}
        >
          <div
            className="sessions-section-mockup__modal"
            onClick={(e) => e.stopPropagation()}
          >
            <h3>New Session</h3>
            <p>A new session would be created here.</p>
            <button onClick={() => setShowNewSessionModal(false)}>Close</button>
          </div>
        </div>
      )}
    </div>
  );
}
