// Package opencode provides an adapter for the OpenCode agent server,
// converting its native event format to standard ACP messages.
package opencode

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ProxySSE streams SSE from the opencode server to the client,
// converting OpenCode events to standard ACP events.
func ProxySSE(w http.ResponseWriter, r *http.Request, port int) {
	targetURL := fmt.Sprintf("http://127.0.0.1:%d%s", port, r.URL.Path)
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	req, err := http.NewRequestWithContext(r.Context(), "GET", targetURL, nil)
	if err != nil {
		http.Error(w, "failed to create request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{Timeout: 0}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "failed to connect to agent server", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(resp.StatusCode)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			fmt.Fprintf(w, "%s\n", line)
			flusher.Flush()
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		acpEvent := convertSSEEventToACP(data)
		if acpEvent != "" {
			fmt.Fprintf(w, "data: %s\n", acpEvent)
		} else {
			fmt.Fprintf(w, "%s\n", line)
		}
		flusher.Flush()
	}
}

// ProxyMessages fetches messages from the opencode server,
// converts them to ACP format, and writes them to the response.
func ProxyMessages(w http.ResponseWriter, r *http.Request, port int) {
	targetURL := fmt.Sprintf("http://127.0.0.1:%d%s", port, r.URL.Path)
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	req, err := http.NewRequestWithContext(r.Context(), "GET", targetURL, nil)
	if err != nil {
		http.Error(w, "failed to create request", http.StatusInternalServerError)
		return
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "failed to connect to agent server", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	var rawMessages []json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&rawMessages); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}

	acpMessages := make([]map[string]interface{}, 0, len(rawMessages))
	for _, raw := range rawMessages {
		acpMsg := convertMessageToACP(raw)
		if acpMsg != nil {
			acpMessages = append(acpMessages, acpMsg)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(acpMessages)
}
