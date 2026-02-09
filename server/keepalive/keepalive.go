package keepalive

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
)

// RegisterAPI registers the keep-alive proxy and status endpoints.
func RegisterAPI(mux *http.ServeMux) {
	// This endpoint checks whether the keep-alive daemon is running
	// by probing the keep-alive port. It does NOT proxy.
	mux.HandleFunc("/api/keep-alive/ping", handleKeepAlivePing)

	// All other /api/keep-alive/* requests are proxied to the keep-alive server.
	targetURL, _ := url.Parse(fmt.Sprintf("http://localhost:%d", config.KeepAlivePort))
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	mux.HandleFunc("/api/keep-alive/", func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	})
}

func handleKeepAlivePing(w http.ResponseWriter, r *http.Request) {
	running := isKeepAliveRunning()

	resp := map[string]interface{}{
		"running": running,
	}

	if !running {
		// Provide the start command hint
		binPath, err := os.Executable()
		if err == nil {
			resp["start_command"] = fmt.Sprintf("%s keep-alive", binPath)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func isKeepAliveRunning() bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", config.KeepAlivePort), 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
