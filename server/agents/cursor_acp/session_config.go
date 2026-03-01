package cursor_acp

// SessionConfig holds per-session configuration for cursor agent
type SessionConfig struct {
	ID string `json:"id"`
	// TrustWorkspace indicates if the workspace is trusted
	TrustWorkspace bool `json:"trustWorkspace,omitempty"`
	// YoloMode indicates if --yolo flag should be passed (bypass all confirmations)
	YoloMode bool `json:"yoloMode,omitempty"`
}

// SessionConfigStore provides persistent storage for session configurations
type SessionConfigStore struct {
	store map[string]*SessionConfig
}

func NewSessionConfigStore() *SessionConfigStore {
	return &SessionConfigStore{
		store: make(map[string]*SessionConfig),
	}
}

func (s *SessionConfigStore) Get(sessionID string) *SessionConfig {
	if cfg, ok := s.store[sessionID]; ok {
		return cfg
	}
	return nil
}

func (s *SessionConfigStore) Set(sessionID string, cfg *SessionConfig) {
	cfg.ID = sessionID
	s.store[sessionID] = cfg
}

func (s *SessionConfigStore) UpdateTrust(sessionID string, trust bool) {
	if cfg, ok := s.store[sessionID]; ok {
		cfg.TrustWorkspace = trust
	} else {
		s.store[sessionID] = &SessionConfig{
			ID:             sessionID,
			TrustWorkspace: trust,
		}
	}
}

func (s *SessionConfigStore) UpdateYoloMode(sessionID string, yolo bool) {
	if cfg, ok := s.store[sessionID]; ok {
		cfg.YoloMode = yolo
	} else {
		s.store[sessionID] = &SessionConfig{
			ID:       sessionID,
			YoloMode: yolo,
		}
	}
}
