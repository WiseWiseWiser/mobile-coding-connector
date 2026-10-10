package qemu

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/xhd2015/ai-critic/server/streaming/progress"
)

const guestProbeTimeout = 15 * time.Second

func handleGuestStatusStream(w http.ResponseWriter, r *http.Request) {
	streamGuestProbe(w, r, false)
}

func handleGuestCFStatusStream(w http.ResponseWriter, r *http.Request) {
	streamGuestProbe(w, r, true)
}

func streamGuestProbe(w http.ResponseWriter, r *http.Request, cloudflared bool) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeActionErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	pw := progress.NewWriter(w)
	if pw == nil {
		writeActionErr(w, http.StatusInternalServerError, "streaming not supported")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), guestProbeTimeout)
	defer cancel()
	sawCF := false
	onLine := func(line string) {
		if strings.HasPrefix(line, "cf_alive=") {
			sawCF = true
		}
		_ = pw.EmitLog(line, true)
	}
	m := DefaultManager()
	var err error
	if cloudflared {
		_, err = m.CFStatusLines(ctx, "", onLine)
	} else {
		_, err = m.StatusLines(ctx, onLine)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		_ = pw.EmitLog("warning: guest ssh timed out after 15s", true)
		if cloudflared && !sawCF {
			_ = pw.EmitLog("cf_alive=unknown", true)
		}
		_ = pw.EmitDone(map[string]any{"ok": true, "timed_out": true})
		return
	}
	if err != nil {
		_ = pw.EmitError(err.Error())
		return
	}
	_ = pw.EmitDone(map[string]any{"ok": true})
}
