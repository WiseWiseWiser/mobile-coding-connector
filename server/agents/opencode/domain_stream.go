package opencode

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/portforward"
)

// StreamSession represents an active domain mapping streaming session
type StreamSession struct {
	ID        string
	Port      int
	Domain    string
	Provider  string
	Status    string
	Logs      []LogEntry
	PublicURL string
	Error     string
	Done      bool
	Success   bool
	CreatedAt time.Time
	UpdatedAt time.Time
	mu        sync.RWMutex
	doneChan  chan struct{}
}

// LogEntry represents a single log line with timestamp
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	IsError   bool      `json:"is_error"`
}

// SessionManager manages active streaming sessions
type SessionManager struct {
	sessions map[string]*StreamSession
	mu       sync.RWMutex
}

var (
	sessionManager     *SessionManager
	sessionManagerOnce sync.Once
)

// GetSessionManager returns the global session manager singleton
func GetSessionManager() *SessionManager {
	sessionManagerOnce.Do(func() {
		sessionManager = &SessionManager{
			sessions: make(map[string]*StreamSession),
		}
	})
	return sessionManager
}

// CreateSession creates a new streaming session
func (sm *SessionManager) CreateSession(id string, port int, domain string, provider string) *StreamSession {
	session := &StreamSession{
		ID:        id,
		Port:      port,
		Domain:    domain,
		Provider:  provider,
		Status:    "starting",
		Logs:      make([]LogEntry, 0),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		doneChan:  make(chan struct{}),
	}

	sm.mu.Lock()
	sm.sessions[id] = session
	sm.mu.Unlock()

	// Clean up old sessions after 1 hour
	go sm.cleanupOldSessions()

	return session
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(id string) (*StreamSession, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	session, ok := sm.sessions[id]
	return session, ok
}

// cleanupOldSessions removes sessions older than 1 hour
func (sm *SessionManager) cleanupOldSessions() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	cutoff := time.Now().Add(-1 * time.Hour)
	for id, session := range sm.sessions {
		if session.CreatedAt.Before(cutoff) {
			delete(sm.sessions, id)
		}
	}
}

// AddLog adds a log entry to the session
func (s *StreamSession) AddLog(message string, isError bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Logs = append(s.Logs, LogEntry{
		Timestamp: time.Now(),
		Message:   message,
		IsError:   isError,
	})
	s.UpdatedAt = time.Now()
}

// SetStatus updates the session status
func (s *StreamSession) SetStatus(status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = status
	s.UpdatedAt = time.Now()
}

// SetResult sets the final result and marks session as done
func (s *StreamSession) SetResult(success bool, publicURL string, err string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Done = true
	s.Success = success
	s.PublicURL = publicURL
	s.Error = err
	s.UpdatedAt = time.Now()
	close(s.doneChan)
}

// GetLogsSince returns logs after a specific index
func (s *StreamSession) GetLogsSince(startIndex int) []LogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if startIndex >= len(s.Logs) {
		return []LogEntry{}
	}
	return s.Logs[startIndex:]
}

// WaitDone returns a channel that closes when the session is done
func (s *StreamSession) WaitDone() <-chan struct{} {
	return s.doneChan
}

// IsDone checks if the session is complete
func (s *StreamSession) IsDone() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Done
}

// MapDomainViaCloudflareStreaming starts a domain mapping operation with streaming support
// Returns a session ID that can be used to reconnect and get updates
func MapDomainViaCloudflareStreaming(provider string, sessionID string) (*StreamSession, error) {
	settings, err := LoadSettings()
	if err != nil {
		return nil, err
	}

	// Create or get session
	manager := GetSessionManager()
	var session *StreamSession

	if sessionID != "" {
		if existingSession, ok := manager.GetSession(sessionID); ok {
			session = existingSession
		}
	}

	if session == nil {
		// Create new session
		if sessionID == "" {
			sessionID = fmt.Sprintf("domain-map-%d", time.Now().UnixNano())
		}

		// Validate settings
		if settings.DefaultDomain == "" {
			session = manager.CreateSession(sessionID, 0, "", provider)
			session.AddLog("Error: No default domain configured", true)
			session.SetResult(false, "", "No default domain configured")
			return session, nil
		}

		if !IsWebServerRunning(settings.WebServer.Port) {
			session = manager.CreateSession(sessionID, settings.WebServer.Port, settings.DefaultDomain, provider)
			session.AddLog("Error: Web server is not running", true)
			session.SetResult(false, "", "Web server is not running")
			return session, nil
		}

		// Check if domain matches an owned domain
		matches, _ := DomainMatchesOwned(settings.DefaultDomain)
		if !matches {
			session = manager.CreateSession(sessionID, settings.WebServer.Port, settings.DefaultDomain, provider)
			session.AddLog(fmt.Sprintf("Error: Domain %s does not match any owned domain", settings.DefaultDomain), true)
			session.SetResult(false, "", fmt.Sprintf("Domain %s does not match any owned domain", settings.DefaultDomain))
			return session, nil
		}

		// Default to cloudflare_owned if not specified
		if provider == "" {
			provider = portforward.ProviderCloudflareOwned
		}

		session = manager.CreateSession(sessionID, settings.WebServer.Port, settings.DefaultDomain, provider)

		// Start the mapping process in background
		go runDomainMapping(session, settings, provider)
	}

	return session, nil
}

// runDomainMapping performs the actual domain mapping with progress logging
func runDomainMapping(session *StreamSession, settings *Settings, provider string) {
	session.AddLog(fmt.Sprintf("Starting domain mapping for %s via %s...", session.Domain, provider), false)
	session.SetStatus("mapping")

	// Create a label for this port forward
	label := session.Domain

	// Subscribe to port forward changes to get status updates
	pfManager := portforward.GetDefaultManager()
	subID, subChan := pfManager.Subscribe()
	defer pfManager.Unsubscribe(subID)

	// Start the port forward
	session.AddLog("Creating Cloudflare tunnel...", false)
	pf, err := pfManager.Add(session.Port, label, provider)
	if err != nil {
		session.AddLog(fmt.Sprintf("Failed to create port forward: %v", err), true)
		session.SetResult(false, "", fmt.Sprintf("Failed to create port forward: %v", err))
		return
	}

	session.AddLog(fmt.Sprintf("Port forward created with status: %s", pf.Status), false)

	// Monitor the port forward until it's active or errors
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			session.AddLog("Timeout waiting for tunnel to become active", true)
			session.SetResult(false, "", "Timeout waiting for tunnel to become active")
			return

		case <-session.WaitDone():
			// Session was already completed (maybe cancelled)
			return

		case ports, ok := <-subChan:
			if !ok {
				session.AddLog("Port forward subscription closed", true)
				session.SetResult(false, "", "Port forward subscription closed")
				return
			}

			// Find our port forward
			for _, port := range ports {
				if port.LocalPort == session.Port {
					switch port.Status {
					case portforward.StatusActive:
						session.AddLog(fmt.Sprintf("✓ Tunnel is active! Public URL: %s", port.PublicURL), false)
						session.SetStatus("active")

						// Save the exposed domain
						settings.WebServer.ExposedDomain = port.PublicURL
						SaveSettings(settings)

						session.SetResult(true, port.PublicURL, "")
						return

					case portforward.StatusError:
						session.AddLog(fmt.Sprintf("✗ Tunnel failed: %s", port.Error), true)
						session.SetStatus("error")
						session.SetResult(false, "", port.Error)
						return

					case portforward.StatusConnecting:
						session.AddLog("Tunnel is connecting...", false)
						session.SetStatus("connecting")

					case portforward.StatusStopped:
						session.AddLog("Tunnel was stopped", true)
						session.SetStatus("stopped")
						session.SetResult(false, "", "Tunnel was stopped")
						return
					}
					break
				}
			}
		}
	}
}

// GetSessionLogs returns all logs from a session starting from a given index
func GetSessionLogs(sessionID string, startIndex int) ([]LogEntry, bool, error) {
	manager := GetSessionManager()
	session, ok := manager.GetSession(sessionID)
	if !ok {
		return nil, false, fmt.Errorf("session not found: %s", sessionID)
	}

	logs := session.GetLogsSince(startIndex)
	return logs, session.IsDone(), nil
}
