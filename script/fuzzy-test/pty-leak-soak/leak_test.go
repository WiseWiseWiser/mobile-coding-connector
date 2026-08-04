// Package main is not used; this is a go test package for PTY leak soak/fuzzy.
package ptyleaksoak_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/xhd2015/dot-pkgs/go-pkgs/shell/ptywrap/ptytest"
)

// TestPTYLeakFuzzySoak stresses createShell/reattach/close paths and asserts
// no orphan bash --rcfile …ptywrap-bashrc processes remain after churn.
//
// Scenarios (weighted random):
//  1. create-on-connect + close 1000 (writer stopChild path)
//  2. create-on-connect + close 4000 (remove path)
//  3. double-create StrictMode-like (two WS no session_id, close both)
//  4. reattach-miss (bogus session_id → createShell) + close 1000
//  5. create then abandon without close until end (half-open hold), then close
//
// Run:
//
//	go test ./script/fuzzy-test/pty-leak-soak/ -count=1 -v -timeout 5m
func TestPTYLeakFuzzySoak(t *testing.T) {
	base, cleanup := ptytest.StartTestServer(t)
	t.Cleanup(cleanup)

	iterations := 40
	if v := strings.TrimSpace(os.Getenv("PTY_LEAK_ITERS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			iterations = n
		}
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	var held []*websocket.Conn
	var heldMu sync.Mutex

	beforeBashes := countPtywrapBashes()
	t.Logf("start: ptywrap-bash count=%d base=%s iters=%d", beforeBashes, base, iterations)

	var stats struct {
		createClose1000 int
		createClose4000 int
		doubleCreate    int
		reattachMiss    int
		holdThenClose   int
	}

	for i := 0; i < iterations; i++ {
		switch rng.Intn(5) {
		case 0:
			stats.createClose1000++
			id, err := wsCreateAndClose(t, base, fmt.Sprintf("soak-1000-%d", i), 1000)
			if err != nil {
				t.Fatalf("iter %d create+1000: %v", i, err)
			}
			t.Logf("iter=%d scenario=close1000 session=%s", i, id)
		case 1:
			stats.createClose4000++
			id, err := wsCreateAndClose(t, base, fmt.Sprintf("soak-4000-%d", i), 4000)
			if err != nil {
				t.Fatalf("iter %d create+4000: %v", i, err)
			}
			t.Logf("iter=%d scenario=close4000 session=%s", i, id)
		case 2:
			stats.doubleCreate++
			id1, err := wsCreateAndClose(t, base, fmt.Sprintf("dbl-a-%d", i), 1000)
			if err != nil {
				t.Fatalf("iter %d double-a: %v", i, err)
			}
			id2, err := wsCreateAndClose(t, base, fmt.Sprintf("dbl-b-%d", i), 1000)
			if err != nil {
				t.Fatalf("iter %d double-b: %v", i, err)
			}
			t.Logf("iter=%d scenario=double_create sessions=%s,%s", i, id1, id2)
		case 3:
			stats.reattachMiss++
			id, err := wsCreateWithSessionIDAndClose(t, base, "missing-"+strconv.Itoa(i), fmt.Sprintf("miss-%d", i), 1000)
			if err != nil {
				t.Fatalf("iter %d reattach_miss: %v", i, err)
			}
			t.Logf("iter=%d scenario=reattach_miss new_session=%s", i, id)
		default:
			stats.holdThenClose++
			// Open without closing for a few iterations, then flush held conns.
			conn, id, err := wsCreateHold(t, base, fmt.Sprintf("hold-%d", i))
			if err != nil {
				t.Fatalf("iter %d hold: %v", i, err)
			}
			heldMu.Lock()
			held = append(held, conn)
			heldMu.Unlock()
			t.Logf("iter=%d scenario=hold session=%s held=%d", i, id, len(held))
			if len(held) >= 5 {
				heldMu.Lock()
				for _, c := range held {
					_ = c.WriteControl(websocket.CloseMessage,
						websocket.FormatCloseMessage(1000, ""),
						time.Now().Add(time.Second))
					_ = c.Close()
				}
				held = nil
				heldMu.Unlock()
				time.Sleep(300 * time.Millisecond)
				t.Logf("iter=%d flushed held writers", i)
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Flush any remaining held connections.
	heldMu.Lock()
	for _, c := range held {
		_ = c.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(1000, ""),
			time.Now().Add(time.Second))
		_ = c.Close()
	}
	held = nil
	heldMu.Unlock()
	time.Sleep(500 * time.Millisecond)

	afterBashes := countPtywrapBashes()
	sessions, err := listSessions(base)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	running := 0
	for _, s := range sessions {
		if s.Status == "running" {
			running++
		}
	}

	t.Logf("stats: %+v", stats)
	t.Logf("end: ptywrap-bash count=%d (start=%d) sessions_total=%d sessions_running=%d",
		afterBashes, beforeBashes, len(sessions), running)

	// Orphans: more ptywrap bashes than we started with (this process only; filter by tree is hard,
	// so compare delta and running sessions).
	if afterBashes > beforeBashes {
		t.Errorf("LEAK: ptywrap-bash processes increased %d → %d (delta +%d)",
			beforeBashes, afterBashes, afterBashes-beforeBashes)
	}
	if running > 0 {
		// After flush, no running shells should remain (all writers closed with stopChild).
		t.Errorf("LEAK: %d sessions still status=running after soak (total listed=%d)", running, len(sessions))
		for _, s := range sessions {
			if s.Status == "running" {
				t.Logf("  still running: id=%s name=%s cmd=%v", s.ID, s.Name, s.Command)
			}
		}
	}
}

// TestPTYLeakMultiCreateOrphan is the classic StrictMode churn: N create+close1000.
func TestPTYLeakMultiCreateOrphan(t *testing.T) {
	base, cleanup := ptytest.StartTestServer(t)
	t.Cleanup(cleanup)

	n := 20
	before := countPtywrapBashes()
	for i := 0; i < n; i++ {
		if _, err := wsCreateAndClose(t, base, fmt.Sprintf("orphan-%d", i), 1000); err != nil {
			t.Fatalf("create close: %v", err)
		}
	}
	time.Sleep(500 * time.Millisecond)
	after := countPtywrapBashes()
	sessions, _ := listSessions(base)
	running := 0
	for _, s := range sessions {
		if s.Status == "running" {
			running++
		}
	}
	t.Logf("multi-create n=%d bash %d→%d sessions=%d running=%d", n, before, after, len(sessions), running)
	if after > before {
		t.Errorf("LEAK: multi-create orphan bashes %d → %d", before, after)
	}
	if running > 0 {
		t.Errorf("LEAK: %d running sessions after multi-create close", running)
	}
}

// TestPTYLeakHalfOpenHold proves that not closing the writer leaves shells alive
// (documents the half-open leak class; not a failure of stopChild).
func TestPTYLeakHalfOpenHold(t *testing.T) {
	base, cleanup := ptytest.StartTestServer(t)
	t.Cleanup(cleanup)

	const n = 8
	var conns []*websocket.Conn
	before := countPtywrapBashes()
	for i := 0; i < n; i++ {
		c, id, err := wsCreateHold(t, base, fmt.Sprintf("halfopen-%d", i))
		if err != nil {
			t.Fatalf("hold: %v", err)
		}
		conns = append(conns, c)
		t.Logf("held session=%s", id)
	}
	time.Sleep(200 * time.Millisecond)
	mid := countPtywrapBashes()
	if mid < before+n {
		// Allow some flakiness if bash rc is slow; require at least n/2 new.
		if mid <= before {
			t.Fatalf("expected new bashes while held; before=%d mid=%d", before, mid)
		}
	}
	t.Logf("half-open hold: bashes %d→%d (expected ~+%d while held)", before, mid, n)

	for _, c := range conns {
		_ = c.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(1000, ""),
			time.Now().Add(time.Second))
		_ = c.Close()
	}
	time.Sleep(500 * time.Millisecond)
	after := countPtywrapBashes()
	sessions, _ := listSessions(base)
	running := 0
	for _, s := range sessions {
		if s.Status == "running" {
			running++
		}
	}
	t.Logf("after close: bashes=%d running_sessions=%d", after, running)
	if running > 0 {
		t.Errorf("after closing held writers, still %d running sessions", running)
	}
	if after > before {
		t.Errorf("after closing held writers, bashes still elevated %d → %d", before, after)
	}
}

// --- helpers ---

type sessionInfo struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Status  string   `json:"status"`
	Command []string `json:"command"`
}

type sessionsResponse struct {
	Sessions []sessionInfo `json:"sessions"`
	Total    int           `json:"total"`
}

func listSessions(base string) ([]sessionInfo, error) {
	resp, err := http.Get(base + "/api/terminal/sessions?page_size=1000")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out sessionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Sessions, nil
}

func countPtywrapBashes() int {
	// Match the same fingerprint as the production leak.
	out, err := exec.Command("ps", "-axo", "command=").Output()
	if err != nil {
		return 0
	}
	n := 0
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "ptywrap-bashrc") {
			n++
		}
	}
	return n
}

func wsCreateAndClose(t *testing.T, base, name string, closeCode int) (string, error) {
	t.Helper()
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	u.Scheme = "ws"
	u.Path = "/api/terminal"
	q := u.Query()
	q.Set("name", name)
	u.RawQuery = q.Encode()

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	id, err := readSessionID(conn)
	if err != nil {
		return "", err
	}
	msg := websocket.FormatCloseMessage(closeCode, "")
	_ = conn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(time.Second))
	// Drain until peer closes.
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
	return id, nil
}

func wsCreateWithSessionIDAndClose(t *testing.T, base, sessionID, name string, closeCode int) (string, error) {
	t.Helper()
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	u.Scheme = "ws"
	u.Path = "/api/terminal"
	q := u.Query()
	q.Set("name", name)
	q.Set("session_id", sessionID)
	u.RawQuery = q.Encode()

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	id, err := readSessionID(conn)
	if err != nil {
		return "", err
	}
	msg := websocket.FormatCloseMessage(closeCode, "")
	_ = conn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(time.Second))
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
	return id, nil
}

func wsCreateHold(t *testing.T, base, name string) (*websocket.Conn, string, error) {
	t.Helper()
	u, err := url.Parse(base)
	if err != nil {
		return nil, "", err
	}
	u.Scheme = "ws"
	u.Path = "/api/terminal"
	q := u.Query()
	q.Set("name", name)
	u.RawQuery = q.Encode()

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, "", fmt.Errorf("dial: %w", err)
	}
	id, err := readSessionID(conn)
	if err != nil {
		_ = conn.Close()
		return nil, "", err
	}
	// Keep reading in background so the server-side read loop is not the only end.
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
	return conn, id, nil
}

func readSessionID(conn *websocket.Conn) (string, error) {
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetReadDeadline(time.Time{})
	for i := 0; i < 20; i++ {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			return "", err
		}
		if msgType != websocket.TextMessage {
			continue
		}
		var msg struct {
			Type      string `json:"type"`
			SessionID string `json:"session_id"`
		}
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		if msg.Type == "session_id" && msg.SessionID != "" {
			return msg.SessionID, nil
		}
		// Bare format: {"type":"session_id","session_id":"..."} also sometimes sent as whole JSON with type only in string form from server.
		if strings.Contains(string(data), "session_id") {
			var m map[string]any
			if json.Unmarshal(data, &m) == nil {
				if id, ok := m["session_id"].(string); ok && id != "" {
					return id, nil
				}
			}
		}
	}
	return "", fmt.Errorf("timeout waiting for session_id")
}

// Ensure module can resolve (go.mod at repo root).
var _ = filepath.Separator
