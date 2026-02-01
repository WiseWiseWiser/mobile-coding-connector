package ai

// Provider represents an AI provider type
type Provider string

const (
	ProviderOpenAI Provider = "openai"
)

// ChunkType represents the type of a streamed chunk
type ChunkType string

const (
	ChunkTypeThinking ChunkType = "thinking"
	ChunkTypeContent  ChunkType = "content"
	ChunkTypeDone     ChunkType = "done"
	ChunkTypeError    ChunkType = "error"
)

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`    // "user", "assistant", "system"
	Content string `json:"content"` // The message content
}

// StreamChunk represents a chunk of streamed AI response
type StreamChunk struct {
	Type       ChunkType   `json:"type"`                 // "thinking", "content", "done", "error"
	Content    string      `json:"content,omitempty"`    // The actual content
	Error      string      `json:"error,omitempty"`      // Error message if type is "error"
	TokenUsage *TokenUsage `json:"tokenUsage,omitempty"` // Token usage (sent with done chunk)
}

// StreamCallback is called for each chunk of the streamed response
type StreamCallback func(chunk StreamChunk) error

// Config holds AI provider configuration
type Config struct {
	Provider  Provider `json:"provider"`
	APIKey    string   `json:"api_key"`
	BaseURL   string   `json:"base_url,omitempty"`
	Model     string   `json:"model,omitempty"`
	MaxTokens int      `json:"max_tokens,omitempty"`
}

// TokenUsage represents token usage statistics
type TokenUsage struct {
	PromptTokens     int `json:"promptTokens,omitempty"`
	CompletionTokens int `json:"completionTokens,omitempty"`
	TotalTokens      int `json:"totalTokens,omitempty"`
}
