package daemon

import (
	"fmt"
	"net/http"
	"os/exec"
	"strconv"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
	"github.com/xhd2015/lifelog-private/ai-critic/server/sse"
)

// StreamLogs streams the server log via tail -fn100
func StreamLogs(w http.ResponseWriter, r *http.Request) {
	sw := sse.NewWriter(w)
	if sw == nil {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	linesStr := r.URL.Query().Get("lines")
	maxLines := "100"
	if linesStr != "" {
		if n, err := strconv.Atoi(linesStr); err == nil && n > 0 {
			maxLines = strconv.Itoa(n)
		}
	}

	logPath := config.ServerLogFile
	cmd := exec.Command("tail", "-fn"+maxLines, logPath)

	// Kill tail when the client disconnects
	ctx := r.Context()
	go func() {
		<-ctx.Done()
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	if err := sw.StreamCmd(cmd); err != nil {
		sw.SendError(fmt.Sprintf("tail error: %v", err))
	}
	// tail -f runs indefinitely until killed
}
