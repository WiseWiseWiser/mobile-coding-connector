package portforward

import (
	"io"
	"net"
	"net/http"
	"time"
)

// tunnelConnect handles an HTTP CONNECT request inside the ad-hoc visit hop:
// instead of reverse-proxying, it hijacks the client connection, dials the
// mapping's origin target and pipes raw bytes both ways. This is what lets a
// mapped hostname act as an HTTP proxy hop for clients that speak CONNECT —
// e.g. a phone using the mapping domain as its proxy toward an upstream proxy
// service on another host.
//
// Note: cloudflared's HTTP ingress forwards CONNECT only when the edge
// protocol allows it; LAN/direct reachability of the hop is the reliable path.
func tunnelConnect(w http.ResponseWriter, r *http.Request, dialAddr string) {
	backend, err := net.DialTimeout("tcp", dialAddr, 10*time.Second)
	if err != nil {
		http.Error(w, "dial failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer backend.Close()

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijack unsupported", http.StatusInternalServerError)
		return
	}
	client, buf, err := hijacker.Hijack()
	if err != nil {
		return
	}
	defer client.Close()

	if _, err := client.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n")); err != nil {
		return
	}

	go func() {
		_, _ = io.Copy(backend, buf)
		if c, ok := backend.(*net.TCPConn); ok {
			_ = c.CloseWrite()
		}
	}()
	_, _ = io.Copy(client, backend)
}
