// Package wskeepalive sends periodic WebSocket pings so long-lived connections
// survive hops that reap idle sockets.
//
// A visitor WebSocket crosses Cloudflare, which closes a proxied WebSocket that
// carries no traffic for roughly two minutes. A handler that only writes when it
// has something to say therefore goes silent on an idle connection and gets
// dropped, so every long-lived server-side WebSocket must ping on its own.
package wskeepalive

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// DefaultInterval is how often a server pings a WebSocket. It sits well below
// Cloudflare's roughly two minute idle reaper, and matches the cadence the
// agent-run and upload sockets already use.
const DefaultInterval = 30 * time.Second

// writeTimeout bounds one control-frame write.
const writeTimeout = 5 * time.Second

// Start pings conn every interval until stop is called or a ping fails. Pings
// are control frames, so they interleave safely with the handler's data writes,
// and every compliant client (browsers included) answers with a pong. That
// answer is what keeps traffic flowing in both directions, which is what the
// idle reaper actually watches.
func Start(conn *websocket.Conn, interval time.Duration) (stop func()) {
	if conn == nil {
		return func() {}
	}
	if interval <= 0 {
		interval = DefaultInterval
	}
	done := make(chan struct{})
	var once sync.Once
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeTimeout)); err != nil {
					return
				}
			}
		}
	}()
	return func() { once.Do(func() { close(done) }) }
}
