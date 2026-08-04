package ptyleaksoak_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/xhd2015/dot-pkgs/go-pkgs/shell/ptywrap/ptytest"
)

// ---------------------------------------------------------------------------
// Leak theories (what can still produce live bash --rcfile …ptywrap-bashrc)
//
// Already disproved as "stopChild broken on clean close":
//   - create + close 1000/4000 reaps children (see leak_test.go)
//
// Remaining theories under test here:
//
//  T1 REST orphan: POST /sessions with empty command, never attach as writer.
//     stopChild never runs → bash lives until DELETE.
//
//  T2 Writer hold / reconnect storm: open N writers without closing, each
//     create-on-connect. Models browser/tunnel half-open + new tab each time.
//     Bashes accumulate = N while held.
//
//  T3 Hard TCP drop (NetConn.Close without WS close frame): does server still
//     reap? If yes, only pure-idle holds leak; if no, crash without close leaks.
//
//  T4 detach_keep then WS close: intentional keep-child path leaves bash alive.
//
//  T5 Observer-only attach (attach_mode=observer) on a REST-created shell:
//     never claims writer → detach never stopChild.
//
//  T6 reattach_miss storm with previous writers still held: each miss creates
//     another shell; old ones stay if not closed.
//
//  T7 Fuzzy mix of T1–T6 for 60 rounds; report which left running shells.
// ---------------------------------------------------------------------------

func TestTheory1_RESTCreateNeverAttach(t *testing.T) {
	base, cleanup := ptytest.StartTestServer(t)
	t.Cleanup(cleanup)

	before := countPtywrapBashes()
	const n = 10
	var ids []string
	for i := 0; i < n; i++ {
		id, err := restCreateShell(base, fmt.Sprintf("rest-orphan-%d", i), "")
		if err != nil {
			t.Fatalf("REST create: %v", err)
		}
		ids = append(ids, id)
	}
	time.Sleep(300 * time.Millisecond)
	after := countPtywrapBashes()
	running := countRunningSessions(t, base)

	t.Logf("T1 REST orphan: bashes %d→%d running_sessions=%d ids=%v", before, after, running, ids)

	if after < before+n && running < n {
		// Some environments may delay bash start; require clear elevation.
		t.Fatalf("T1 expected ~%d live shells; bashes=%d running=%d", n, after-before, running)
	}
	// Confirmed leak class: processes remain without any WS writer.
	if running < n {
		t.Errorf("T1 LEAK CONFIRMED but running count %d < %d", running, n)
	} else {
		t.Logf("T1 LEAK CONFIRMED: REST create without attach leaves %d running shells", running)
	}

	// Cleanup so later tests in package are not polluted (same process, new servers OK).
	for _, id := range ids {
		_ = restDelete(base, id)
	}
	time.Sleep(200 * time.Millisecond)
}

func TestTheory2_ReconnectStormWhileWritersHeld(t *testing.T) {
	base, cleanup := ptytest.StartTestServer(t)
	t.Cleanup(cleanup)

	const n = 15
	before := countPtywrapBashes()
	var conns []*websocket.Conn
	for i := 0; i < n; i++ {
		c, id, err := wsCreateHold(t, base, fmt.Sprintf("storm-%d", i))
		if err != nil {
			t.Fatalf("hold: %v", err)
		}
		conns = append(conns, c)
		t.Logf("T2 held %s", id)
	}
	time.Sleep(200 * time.Millisecond)
	mid := countPtywrapBashes()
	running := countRunningSessions(t, base)
	t.Logf("T2 storm hold: bashes %d→%d running=%d", before, mid, running)

	if mid < before+n/2 || running < n {
		t.Fatalf("T2 expected accumulation while held; bashes delta=%d running=%d want ~%d",
			mid-before, running, n)
	}
	t.Logf("T2 LEAK PATTERN CONFIRMED: %d concurrent writers ⇒ %d bashes (production-shaped)", n, mid-before)

	for _, c := range conns {
		gracefulClose(c, 1000)
	}
	time.Sleep(400 * time.Millisecond)
	after := countPtywrapBashes()
	running2 := countRunningSessions(t, base)
	t.Logf("T2 after close all: bashes=%d running=%d", after, running2)
	if running2 > 0 {
		t.Errorf("T2 after close still running=%d (stopChild failed?)", running2)
	}
	if after > before {
		t.Errorf("T2 after close bashes elevated %d→%d", before, after)
	}
}

func TestTheory3_HardTCPDropWithoutWSClose(t *testing.T) {
	base, cleanup := ptytest.StartTestServer(t)
	t.Cleanup(cleanup)

	const n = 10
	before := countPtywrapBashes()
	for i := 0; i < n; i++ {
		c, id, err := wsCreateHold(t, base, fmt.Sprintf("harddrop-%d", i))
		if err != nil {
			t.Fatalf("hold: %v", err)
		}
		// Abort TCP without websocket CloseMessage (simulates kill -9 client).
		if nc := c.NetConn(); nc != nil {
			_ = nc.Close()
		} else {
			_ = c.Close()
		}
		t.Logf("T3 hard-dropped %s", id)
	}
	time.Sleep(600 * time.Millisecond)
	after := countPtywrapBashes()
	running := countRunningSessions(t, base)
	t.Logf("T3 hard TCP drop: bashes %d→%d running=%d", before, after, running)

	if running > 0 || after > before {
		t.Logf("T3 LEAK CONFIRMED: hard drop left bashes=%d running=%d", after-before, running)
		t.Errorf("hard TCP drop did not fully reap: bashes %d→%d running=%d", before, after, running)
	} else {
		t.Logf("T3 NO LEAK: server observed disconnect and reaped (EOF on ReadMessage)")
	}
}

func TestTheory4_DetachKeepLeavesChild(t *testing.T) {
	base, cleanup := ptytest.StartTestServer(t)
	t.Cleanup(cleanup)

	before := countPtywrapBashes()
	c, id, err := wsCreateHold(t, base, "detach-keep-1")
	if err != nil {
		t.Fatalf("hold: %v", err)
	}
	if err := c.WriteMessage(websocket.TextMessage, []byte(`{"type":"detach_keep"}`)); err != nil {
		t.Fatalf("detach_keep: %v", err)
	}
	gracefulClose(c, 1000)
	time.Sleep(400 * time.Millisecond)

	after := countPtywrapBashes()
	running := countRunningSessions(t, base)
	// Find our session status
	sessions, _ := listSessions(base)
	var st string
	for _, s := range sessions {
		if s.ID == id {
			st = s.Status
		}
	}
	t.Logf("T4 detach_keep: bashes %d→%d running=%d session=%s status=%s",
		before, after, running, id, st)

	if running >= 1 || after > before {
		t.Logf("T4 KEEP CONFIRMED: detach_keep left child alive (by design for tty-watch)")
	} else {
		// Maybe session still listed as running with process already dead, or stopChild still ran.
		t.Logf("T4: child was reaped despite detach_keep (unexpected if feature works)")
		// Not a test failure for "what can leak" — note for diagnosis.
	}
	_ = restDelete(base, id)
	time.Sleep(200 * time.Millisecond)
}

func TestTheory5_ObserverOnRESTShellNeverReaps(t *testing.T) {
	base, cleanup := ptytest.StartTestServer(t)
	t.Cleanup(cleanup)

	id, err := restCreateShell(base, "observer-target", "")
	if err != nil {
		t.Fatalf("REST create: %v", err)
	}
	before := countPtywrapBashes()
	time.Sleep(150 * time.Millisecond)

	// Attach as observer only — must not claim writer / stopChild on close.
	c, err := wsAttachMode(t, base, id, "observer")
	if err != nil {
		t.Fatalf("observer attach: %v", err)
	}
	gracefulClose(c, 1000)
	time.Sleep(400 * time.Millisecond)

	after := countPtywrapBashes()
	running := countRunningSessions(t, base)
	t.Logf("T5 observer on REST shell: bashes %d→%d running=%d session=%s", before, after, running, id)

	if running < 1 && after <= before-1 {
		t.Logf("T5: shell was reaped (unexpected for pure observer path on REST shell)")
	} else {
		t.Logf("T5 LEAK CONFIRMED: REST shell + observer-only attach leaves running shell")
	}
	_ = restDelete(base, id)
	time.Sleep(200 * time.Millisecond)
}

func TestTheory6_ReattachMissStormWithHeldWriters(t *testing.T) {
	base, cleanup := ptytest.StartTestServer(t)
	t.Cleanup(cleanup)

	const held = 5
	const misses = 10
	before := countPtywrapBashes()
	var conns []*websocket.Conn
	for i := 0; i < held; i++ {
		c, _, err := wsCreateHold(t, base, fmt.Sprintf("held-%d", i))
		if err != nil {
			t.Fatalf("hold: %v", err)
		}
		conns = append(conns, c)
	}
	// Each miss creates a new shell then we close only the miss connection.
	for i := 0; i < misses; i++ {
		id, err := wsCreateWithSessionIDAndClose(t, base, "gone-"+strconv.Itoa(i), "miss-"+strconv.Itoa(i), 1000)
		if err != nil {
			t.Fatalf("miss: %v", err)
		}
		t.Logf("T6 reattach_miss created then closed %s", id)
	}
	time.Sleep(300 * time.Millisecond)
	mid := countPtywrapBashes()
	running := countRunningSessions(t, base)
	t.Logf("T6 held=%d + closed misses=%d → bashes=%d running=%d (expect ~%d held)",
		held, misses, mid-before, running, held)

	if running < held {
		t.Errorf("T6 expected ~%d running held shells, got %d", held, running)
	} else {
		t.Logf("T6 PATTERN: held writers keep bashes; miss creates reaped when closed")
	}

	for _, c := range conns {
		gracefulClose(c, 1000)
	}
	time.Sleep(400 * time.Millisecond)
	if countRunningSessions(t, base) > 0 {
		t.Errorf("T6 after releasing held, still running sessions")
	}
}

// TestTheory7_FuzzyMix randomly applies T1–T6 styles and reports residual running shells.
func TestTheory7_FuzzyMixLeakHunt(t *testing.T) {
	base, cleanup := ptytest.StartTestServer(t)
	t.Cleanup(cleanup)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	const rounds = 50
	var held []*websocket.Conn
	var restIDs []string
	before := countPtywrapBashes()

	type counters struct {
		rest, hold, hardDrop, detachKeep, observer, miss int
	}
	var c counters

	for i := 0; i < rounds; i++ {
		switch rng.Intn(6) {
		case 0: // T1
			c.rest++
			id, err := restCreateShell(base, fmt.Sprintf("fz-rest-%d", i), "")
			if err != nil {
				t.Fatalf("rest: %v", err)
			}
			restIDs = append(restIDs, id)
		case 1: // T2 hold
			c.hold++
			conn, _, err := wsCreateHold(t, base, fmt.Sprintf("fz-hold-%d", i))
			if err != nil {
				t.Fatalf("hold: %v", err)
			}
			held = append(held, conn)
		case 2: // T3 hard drop
			c.hardDrop++
			conn, _, err := wsCreateHold(t, base, fmt.Sprintf("fz-drop-%d", i))
			if err != nil {
				t.Fatalf("drop: %v", err)
			}
			if nc := conn.NetConn(); nc != nil {
				_ = nc.Close()
			}
		case 3: // T4 detach_keep
			c.detachKeep++
			conn, id, err := wsCreateHold(t, base, fmt.Sprintf("fz-keep-%d", i))
			if err != nil {
				t.Fatalf("keep: %v", err)
			}
			_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"detach_keep"}`))
			gracefulClose(conn, 1000)
			restIDs = append(restIDs, id) // may still be running; track for cleanup
		case 4: // T5 observer on new REST shell
			c.observer++
			id, err := restCreateShell(base, fmt.Sprintf("fz-obs-%d", i), "")
			if err != nil {
				t.Fatalf("rest obs: %v", err)
			}
			restIDs = append(restIDs, id)
			if conn, err := wsAttachMode(t, base, id, "observer"); err == nil {
				gracefulClose(conn, 1000)
			}
		default: // T6 miss close
			c.miss++
			_, _ = wsCreateWithSessionIDAndClose(t, base, "fz-missing-"+strconv.Itoa(i), "fz-miss-"+strconv.Itoa(i), 1000)
		}
		if len(held) > 8 {
			// Periodically release some holds so we don't just OOM PTYs.
			for j := 0; j < 3 && len(held) > 0; j++ {
				gracefulClose(held[0], 1000)
				held = held[1:]
			}
		}
	}

	time.Sleep(400 * time.Millisecond)
	runningBeforeCleanup := countRunningSessions(t, base)
	bashesMid := countPtywrapBashes()
	t.Logf("T7 fuzzy mix counters=%+v bashes %d→%d running_before_cleanup=%d held_open=%d rest_tracked=%d",
		c, before, bashesMid, runningBeforeCleanup, len(held), len(restIDs))

	// Residual running is the "possible leak" score for unclean paths.
	if runningBeforeCleanup == 0 && len(held) == 0 {
		t.Logf("T7: no residual running (unexpected if REST/hold paths ran)")
	} else {
		t.Logf("T7 RESIDUAL: %d running sessions with %d held WS still open — leak sources still live",
			runningBeforeCleanup, len(held))
	}

	// Cleanup everything we can so the process ends clean.
	for _, conn := range held {
		gracefulClose(conn, 1000)
	}
	for _, id := range restIDs {
		_ = restDelete(base, id)
	}
	// Delete any remaining running.
	sessions, _ := listSessions(base)
	for _, s := range sessions {
		if s.Status == "running" {
			_ = restDelete(base, s.ID)
		}
	}
	time.Sleep(400 * time.Millisecond)
	finalRun := countRunningSessions(t, base)
	finalBash := countPtywrapBashes()
	t.Logf("T7 after cleanup: running=%d bashes=%d", finalRun, finalBash)
	if finalRun > 0 {
		t.Errorf("T7 cleanup left %d running sessions", finalRun)
	}
	if finalBash > before {
		t.Errorf("T7 cleanup left bashes elevated %d→%d", before, finalBash)
	}
}

// --- theory helpers ---

func restCreateShell(base, name, cwd string) (string, error) {
	body := map[string]any{"name": name}
	if cwd != "" {
		body["cwd"] = cwd
	}
	// empty command → createShell
	raw, _ := json.Marshal(body)
	resp, err := http.Post(base+"/api/terminal/sessions", "application/json", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, data)
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return "", err
	}
	if out.ID == "" {
		return "", fmt.Errorf("no id in %s", data)
	}
	return out.ID, nil
}

func restDelete(base, id string) error {
	req, err := http.NewRequest(http.MethodDelete, base+"/api/terminal/sessions?id="+url.QueryEscape(id), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func countRunningSessions(t *testing.T, base string) int {
	t.Helper()
	sessions, err := listSessions(base)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	n := 0
	for _, s := range sessions {
		if s.Status == "running" {
			n++
		}
	}
	return n
}

func gracefulClose(c *websocket.Conn, code int) {
	if c == nil {
		return
	}
	_ = c.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(code, ""),
		time.Now().Add(time.Second))
	_ = c.Close()
}

func wsAttachMode(t *testing.T, base, sessionID, mode string) (*websocket.Conn, error) {
	t.Helper()
	u, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	u.Scheme = "ws"
	u.Path = "/api/terminal"
	q := u.Query()
	q.Set("session_id", sessionID)
	q.Set("attach_mode", mode)
	u.RawQuery = q.Encode()
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}
	// Drain handshake frames briefly.
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for i := 0; i < 5; i++ {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
	_ = conn.SetReadDeadline(time.Time{})
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
	return conn, nil
}

// silence unused import if strings only used in other file — keep for status checks
var _ = strings.Contains
