package eventbus

import (
	"fmt"
	"strconv"

	sharedeb "github.com/xhd2015/dot-pkgs/go-pkgs/eventbus"
)

// PublishConfig is the resolved publish-listener configuration from CLI flags.
type PublishConfig struct {
	Port     int
	Token    string
	Disabled bool
}

// DefaultPublishPort returns the conventional loopback publish HTTP port (23891).
func DefaultPublishPort() int {
	return sharedeb.DefaultPublishPort
}

// ResolvePublishConfig maps CLI flags to a PublishConfig.
// Zero portFlag resolves to DefaultPublishPort (23891).
// noPublish=true sets Disabled and skips starting the publish listener.
func ResolvePublishConfig(portFlag int, token string, noPublish bool) PublishConfig {
	cfg := PublishConfig{
		Port:     portFlag,
		Token:    token,
		Disabled: noPublish,
	}
	if cfg.Port <= 0 {
		cfg.Port = DefaultPublishPort()
	}
	return cfg
}

// AppendEventBusServerArgs appends keep-alive child argv flags for non-default
// publish settings. Disabled configs add --no-event-bus-publish only.
// Default port with empty token and publish enabled appends nothing.
func AppendEventBusServerArgs(args []string, cfg PublishConfig) []string {
	out := append([]string(nil), args...)
	if cfg.Disabled {
		out = append(out, "--no-event-bus-publish")
		return out
	}
	if cfg.Port > 0 && cfg.Port != DefaultPublishPort() {
		out = append(out, "--event-bus-publish-port", strconv.Itoa(cfg.Port))
	}
	if cfg.Token != "" {
		out = append(out, "--event-bus-publish-token", cfg.Token)
	}
	return out
}

// ListenAddr returns the loopback bind address for cfg.Port.
func (cfg PublishConfig) ListenAddr() string {
	port := cfg.Port
	if port <= 0 {
		port = DefaultPublishPort()
	}
	return fmt.Sprintf("127.0.0.1:%d", port)
}
