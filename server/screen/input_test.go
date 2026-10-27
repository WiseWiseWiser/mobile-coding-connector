package screen

import (
	"encoding/binary"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

// call records one backend invocation so tests can assert the exact injection
// sequence without touching the host.
type call struct {
	name string
	a    int
	b    int
	s    string
}

type fakeBackend struct {
	mu    sync.Mutex
	calls []call
	err   error
	infoV InputInfo
}

func (b *fakeBackend) record(c call) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.calls = append(b.calls, c)
}

// snapshot copies the recorded calls so a test goroutine can inspect them while
// the manager's reader goroutine keeps injecting.
func (b *fakeBackend) snapshot() []call {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]call(nil), b.calls...)
}

func (b *fakeBackend) displays() ([]Display, error) { return nil, nil }
func (b *fakeBackend) pointerPos() (int, int, bool) { return 0, 0, false }
func (b *fakeBackend) movePointer(x, y int) error {
	b.record(call{name: "move", a: x, b: y})
	return b.err
}
func (b *fakeBackend) button(btn byte, down bool) error {
	d := 0
	if down {
		d = 1
	}
	b.record(call{name: "button", a: int(btn), b: d})
	return b.err
}
func (b *fakeBackend) wheel(dx, dy int) error {
	b.record(call{name: "wheel", a: dx, b: dy})
	return b.err
}
func (b *fakeBackend) key(code string, down bool) error {
	d := 0
	if down {
		d = 1
	}
	b.record(call{name: "key", a: d, s: code})
	return b.err
}
func (b *fakeBackend) text(s string) error {
	b.record(call{name: "text", s: s})
	return b.err
}
func (b *fakeBackend) releaseAll() { b.calls = append(b.calls, call{name: "releaseAll"}) }
func (b *fakeBackend) close()      {}
func (b *fakeBackend) info() InputInfo {
	if b.infoV.Backend == "" {
		return InputInfo{Available: true, Permission: "granted", Backend: "fake"}
	}
	return b.infoV
}

func (b *fakeBackend) names() []string {
	calls := b.snapshot()
	out := make([]string, len(calls))
	for i, c := range calls {
		out[i] = c.name
	}
	return out
}

// testManager wires a manager to a fake backend with one 1728x1117 desktop
// whose frames are 1280x828, the same geometry as a Retina MacBook.
func testManager(t *testing.T) (*Manager, *fakeBackend, *viewer) {
	t.Helper()
	be := &fakeBackend{}
	m := &Manager{
		cfg:      DefaultConfig(),
		backend:  be,
		displays: []Display{{ID: "main", W: 1728, H: 1117, Scale: 2, Primary: true}},
		imgW:     1280,
		imgH:     828,
		state:    StateStreaming,
		viewers:  make(map[*viewer]struct{}),
	}
	// startCaptureLocked seeds these in production; housekeeping is edge
	// triggered on them, so a test that calls it directly needs them too.
	m.hostLocked = hostLocked()
	m.inputOK = be.info().Available
	v := newTestViewer("v-test")
	m.viewers[v] = struct{}{}
	m.holder = v
	m.holderSince = time.Now()
	return m, be, v
}

func newTestViewer(id string) *viewer {
	return &viewer{id: id, send: make(chan outMessage, 32), done: make(chan struct{})}
}

func pointerMsg(action, button byte, x, y int, aux int) []byte {
	buf := make([]byte, pointerEventSize)
	buf[0] = msgPointer
	buf[1] = action
	buf[2] = button
	binary.LittleEndian.PutUint16(buf[4:], uint16(x))
	binary.LittleEndian.PutUint16(buf[6:], uint16(y))
	binary.LittleEndian.PutUint16(buf[8:], uint16(int16(aux)))
	return buf
}

func keyMsg(code string, down bool) []byte {
	buf := make([]byte, 4+len(code))
	buf[0] = msgKey
	if down {
		buf[1] = 1
	}
	binary.LittleEndian.PutUint16(buf[2:], uint16(len(code)))
	copy(buf[4:], code)
	return buf
}

func textMsg(s string) []byte {
	buf := make([]byte, 3+len(s))
	buf[0] = msgText
	binary.LittleEndian.PutUint16(buf[1:], uint16(len(s)))
	copy(buf[3:], s)
	return buf
}

// drainOne pops one pending outbound message, or nil when the viewer has none.
func drainOne(v *viewer) *outMessage {
	select {
	case msg := <-v.send:
		return &msg
	default:
		return nil
	}
}

func TestInputMoveMapsToHostCoordinates(t *testing.T) {
	m, be, v := testManager(t)

	m.handleBinary(v, pointerMsg(actionMove, buttonNone, 640, 414, 0))

	if len(be.calls) != 1 {
		t.Fatalf("calls = %v", be.names())
	}
	if got := be.calls[0]; got.name != "move" || got.a != 864 || got.b != 558 {
		t.Fatalf("move = %+v, want 864,558", got)
	}
	if m.lastInput.IsZero() {
		t.Fatal("lastInput must advance on injected input")
	}
}

func TestInputClickPostsDownAndUp(t *testing.T) {
	m, be, v := testManager(t)

	m.handleBinary(v, pointerMsg(actionDown, buttonLeft, 100, 100, 0))
	m.handleBinary(v, pointerMsg(actionUp, buttonLeft, 100, 100, 0))

	want := []call{
		{name: "move", a: 135, b: 134},
		{name: "button", a: int(buttonLeft), b: 1},
		{name: "move", a: 135, b: 134},
		{name: "button", a: int(buttonLeft), b: 0},
	}
	if len(be.calls) != len(want) {
		t.Fatalf("calls = %v", be.calls)
	}
	for i, w := range want {
		if be.calls[i] != w {
			t.Fatalf("call %d = %+v, want %+v", i, be.calls[i], w)
		}
	}
}

func TestInputMiddleAndRightButtons(t *testing.T) {
	m, be, v := testManager(t)

	m.handleBinary(v, pointerMsg(actionDown, buttonMiddle, 10, 10, 0))
	m.handleBinary(v, pointerMsg(actionUp, buttonMiddle, 10, 10, 0))
	m.handleBinary(v, pointerMsg(actionDown, buttonRight, 10, 10, 0))
	m.handleBinary(v, pointerMsg(actionUp, buttonRight, 10, 10, 0))

	var buttons []int
	for _, c := range be.calls {
		if c.name == "button" {
			buttons = append(buttons, c.a)
		}
	}
	want := []int{int(buttonMiddle), int(buttonMiddle), int(buttonRight), int(buttonRight)}
	if len(buttons) != len(want) {
		t.Fatalf("buttons = %v", buttons)
	}
	for i := range want {
		if buttons[i] != want[i] {
			t.Fatalf("buttons = %v, want %v", buttons, want)
		}
	}
}

func TestInputWheelPassesDelta(t *testing.T) {
	m, be, v := testManager(t)

	m.handleBinary(v, pointerMsg(actionWheel, buttonNone, 50, 50, -3))

	if len(be.calls) != 1 || be.calls[0].name != "wheel" || be.calls[0].b != -3 {
		t.Fatalf("calls = %+v", be.calls)
	}
}

func TestInputKeyAndText(t *testing.T) {
	m, be, v := testManager(t)

	m.handleBinary(v, keyMsg("KeyA", true))
	m.handleBinary(v, keyMsg("KeyA", false))
	m.handleBinary(v, textMsg("hi ✓"))

	want := []call{
		{name: "key", a: 1, s: "KeyA"},
		{name: "key", a: 0, s: "KeyA"},
		{name: "text", s: "hi ✓"},
	}
	if len(be.calls) != len(want) {
		t.Fatalf("calls = %+v", be.calls)
	}
	for i, w := range want {
		if be.calls[i] != w {
			t.Fatalf("call %d = %+v, want %+v", i, be.calls[i], w)
		}
	}
}

// A locked host must be driven by nobody: injected keys would land in the
// unlock prompt, so control is dropped and further input is refused.
func TestLockedHostReleasesControlAndDropsInput(t *testing.T) {
	m, be, v := testManager(t)
	m.lastInput = time.Now()

	locked := false
	lockProbe = func() bool { return locked }
	t.Cleanup(func() { lockProbe = func() bool { return false } })

	locked = true
	m.housekeeping(m.gen)
	m.mu.Lock()
	holder := m.holder
	m.mu.Unlock()
	if holder != nil {
		t.Fatalf("control must be released once the host locks, still held by %s", holder.id)
	}

	m.handleBinary(v, pointerMsg(actionMove, buttonNone, 10, 10, 0))
	for _, name := range be.names() {
		// releaseAll is expected: dropping control must not leave a key down.
		if name != "releaseAll" {
			t.Fatalf("input reached the backend while locked: %v", be.names())
		}
	}
	var sawLocked bool
	for drained := false; !drained; {
		select {
		case msg := <-v.send:
			var payload map[string]any
			if err := json.Unmarshal(msg.data, &payload); err == nil && payload["code"] == "host_locked" {
				sawLocked = true
			}
		default:
			drained = true
		}
	}
	if !sawLocked {
		t.Fatal("viewer was not told the host is locked")
	}

	locked = false
	if ok, reason := m.requestControl(v); !ok {
		t.Fatalf("control must be available again after unlock: %s", reason)
	}
}

func TestInputRejectedWithoutControl(t *testing.T) {
	m, be, v := testManager(t)
	other := newTestViewer("v-other")
	m.viewers[other] = struct{}{}
	m.holder = other

	m.handleBinary(v, pointerMsg(actionDown, buttonLeft, 100, 100, 0))
	m.handleBinary(v, keyMsg("KeyA", true))
	m.handleBinary(v, textMsg("x"))

	if len(be.calls) != 0 {
		t.Fatalf("non-holder must not inject: %v", be.names())
	}
	select {
	case msg := <-v.send:
		var payload struct {
			Type string `json:"type"`
			Code string `json:"code"`
		}
		if err := json.Unmarshal(msg.data, &payload); err != nil {
			t.Fatalf("bad error payload: %v", err)
		}
		if payload.Type != "error" || payload.Code != "not_controller" {
			t.Fatalf("payload = %+v", payload)
		}
	default:
		t.Fatal("expected a not_controller notice")
	}
}

func TestInputDenialIsThrottled(t *testing.T) {
	m, _, v := testManager(t)
	m.holder = newTestViewer("v-other")

	m.handleBinary(v, pointerMsg(actionMove, buttonNone, 1, 1, 0))
	m.handleBinary(v, pointerMsg(actionMove, buttonNone, 2, 2, 0))
	m.handleBinary(v, pointerMsg(actionMove, buttonNone, 3, 3, 0))

	count := 0
	for {
		select {
		case <-v.send:
			count++
			continue
		default:
		}
		break
	}
	if count != 1 {
		t.Fatalf("denial notices = %d, want 1", count)
	}
}

// The text frame shares the denial throttle with pointer frames: a viewer with
// no lock gets one notice per second, not one per attempt. That is the reason
// the client cannot restore a wiped draft from the error channel alone — the
// second attempt inside the window produces no error at all.
func TestTextDenialIsThrottled(t *testing.T) {
	m, be, v := testManager(t)
	m.holder = newTestViewer("v-other")

	m.handleBinary(v, textMsg("hello"))
	m.handleBinary(v, textMsg("hello"))

	if len(be.calls) != 0 {
		t.Fatalf("non-holder must not inject: %v", be.names())
	}

	count := 0
	codes := map[string]int{}
	for {
		msg := drainOne(v)
		if msg == nil {
			break
		}
		var payload struct {
			Type string `json:"type"`
			Code string `json:"code"`
		}
		if err := json.Unmarshal(msg.data, &payload); err != nil {
			t.Fatalf("bad error payload: %v", err)
		}
		codes[payload.Code]++
		count++
	}
	if count != 1 || codes["not_controller"] != 1 {
		t.Fatalf("denial notices = %d (%v), want exactly 1 not_controller", count, codes)
	}
}

// An empty text frame is dropped before it reaches the backend and is not an
// error either: the client already treats "nothing to send" as a local no-op,
// and the host must not see a phantom keystroke for it.
func TestEmptyTextIsSilentlyIgnored(t *testing.T) {
	m, be, v := testManager(t)

	m.handleBinary(v, textMsg(""))

	if len(be.calls) != 0 {
		t.Fatalf("empty text must not inject: %v", be.names())
	}
	if msg := drainOne(v); msg != nil {
		t.Fatalf("empty text must not notify, got %s", msg.data)
	}
}

func TestInputIgnoredWhenControlDisabled(t *testing.T) {
	m, be, v := testManager(t)
	m.cfg.AllowControl = false

	m.handleBinary(v, pointerMsg(actionDown, buttonLeft, 100, 100, 0))
	m.handleBinary(v, keyMsg("KeyA", true))

	if len(be.calls) != 0 {
		t.Fatalf("disabled control must not inject: %v", be.names())
	}
}

func TestInputIgnoredBeforeFirstFrame(t *testing.T) {
	m, be, v := testManager(t)
	m.imgW, m.imgH = 0, 0

	m.handleBinary(v, pointerMsg(actionMove, buttonNone, 10, 10, 0))

	if len(be.calls) != 0 {
		t.Fatalf("no geometry means no injection: %v", be.names())
	}
}

func TestReleaseAllMessageDropsControl(t *testing.T) {
	m, be, v := testManager(t)

	m.handleBinary(v, []byte{msgReleaseAll})

	if m.holder != nil {
		t.Fatal("holder must be cleared")
	}
	found := false
	for _, c := range be.calls {
		if c.name == "releaseAll" {
			found = true
		}
	}
	if !found {
		t.Fatalf("backend.releaseAll not called: %v", be.names())
	}
}

func TestInputRateLimit(t *testing.T) {
	m, be, v := testManager(t)

	for i := 0; i < 500; i++ {
		m.handleBinary(v, pointerMsg(actionMove, buttonNone, 10, 10, 0))
	}

	// The bucket starts full at inputBurst and refills at inputRateLimit/s, so a
	// tight loop may only pass a few extra events.
	if len(be.calls) > inputBurst+10 {
		t.Fatalf("injected %d events, want <= %d", len(be.calls), inputBurst+10)
	}
	if len(be.calls) == 0 {
		t.Fatal("limiter dropped every event")
	}
}

func TestInputErrorIsReported(t *testing.T) {
	m, be, v := testManager(t)
	be.err = errShortMessage

	m.handleBinary(v, pointerMsg(actionMove, buttonNone, 10, 10, 0))

	select {
	case msg := <-v.send:
		var payload struct {
			Type string `json:"type"`
			Code string `json:"code"`
		}
		if err := json.Unmarshal(msg.data, &payload); err != nil {
			t.Fatalf("bad payload: %v", err)
		}
		if payload.Type != "error" || payload.Code != "input_failed" {
			t.Fatalf("payload = %+v", payload)
		}
	default:
		t.Fatal("expected input_failed notice")
	}
}

func TestRequestControlConflict(t *testing.T) {
	m, _, v := testManager(t)
	other := newTestViewer("v-other")
	m.viewers[other] = struct{}{}
	m.holder = other

	granted, reason := m.requestControl(v)
	if granted {
		t.Fatal("second viewer must not take control while locked")
	}
	if reason == "" {
		t.Fatal("expected a reason")
	}
	if m.holder != other {
		t.Fatal("holder must not change")
	}

	// Re-requesting control from the current holder is idempotent.
	if granted, reason := m.requestControl(other); !granted {
		t.Fatalf("holder re-request denied: %s", reason)
	}
}

func TestRequestControlRequiresAvailableBackend(t *testing.T) {
	m, be, _ := testManager(t)
	be.infoV = InputInfo{Available: false, Reason: "no permission", Backend: "fake"}
	v := newTestViewer("v-new")
	m.viewers[v] = struct{}{}
	m.holder = nil

	granted, reason := m.requestControl(v)
	if granted || reason != "no permission" {
		t.Fatalf("granted=%v reason=%q", granted, reason)
	}
}

func TestReleaseControlLockedUnlocksEveryone(t *testing.T) {
	m, be, v := testManager(t)
	other := newTestViewer("v-other")
	m.viewers[other] = struct{}{}
	drain := func(vw *viewer) []map[string]any {
		var out []map[string]any
		for {
			select {
			case msg := <-vw.send:
				var payload map[string]any
				if err := json.Unmarshal(msg.data, &payload); err == nil {
					out = append(out, payload)
				}
				continue
			default:
			}
			return out
		}
	}

	m.mu.Lock()
	m.releaseControlLocked("test")
	m.mu.Unlock()

	if m.holder != nil {
		t.Fatal("holder must be nil after release")
	}
	if len(be.calls) != 1 || be.calls[0].name != "releaseAll" {
		t.Fatalf("calls = %v", be.names())
	}
	for _, vw := range []*viewer{v, other} {
		msgs := drain(vw)
		if len(msgs) == 0 {
			t.Fatalf("viewer %s was not notified", vw.id)
		}
		if msgs[0]["type"] != "control" || msgs[0]["granted"] != false {
			t.Fatalf("viewer %s payload = %v", vw.id, msgs[0])
		}
	}
}

func TestIdleReleaseReleasesStaleControl(t *testing.T) {
	m, _, _ := testManager(t)
	m.lastInput = time.Now().Add(-3 * controlIdleTimeout)

	m.housekeeping(m.gen)

	if m.holder != nil {
		t.Fatal("idle control must be released")
	}
}

func TestActiveHolderKeepsControl(t *testing.T) {
	m, _, _ := testManager(t)
	m.lastInput = time.Now()

	m.housekeeping(m.gen)

	if m.holder == nil {
		t.Fatal("active holder must keep control")
	}
}

func TestHousekeepingLeavesActiveViewersAlone(t *testing.T) {
	m, _, _ := testManager(t)
	m.state = StateStreaming
	m.lastInput = time.Now()

	m.housekeeping(m.gen)

	if m.state != StateStreaming {
		t.Fatalf("state = %s, want streaming", m.state)
	}
}

func TestHousekeepingIgnoresStaleGeneration(t *testing.T) {
	m, _, _ := testManager(t)
	m.gen = 7
	m.lastInput = time.Now().Add(-3 * controlIdleTimeout)

	m.housekeeping(6)

	if m.holder == nil {
		t.Fatal("stale generation must not release control")
	}
}

// A drag is a down, a run of moves and an up. The client sends nothing new for
// it — the backend decides from the held button that those moves are dragged
// events — so this pins the injection order the gesture produces.
func TestInputDragPostsDownMovesAndUp(t *testing.T) {
	m, be, v := testManager(t)

	m.handleBinary(v, pointerMsg(actionDown, buttonLeft, 100, 100, 0))
	m.handleBinary(v, pointerMsg(actionMove, buttonNone, 200, 150, 0))
	m.handleBinary(v, pointerMsg(actionMove, buttonNone, 300, 400, 0))
	m.handleBinary(v, pointerMsg(actionUp, buttonLeft, 300, 400, 0))

	want := []call{
		{name: "move", a: 135, b: 134},
		{name: "button", a: int(buttonLeft), b: 1},
		{name: "move", a: 270, b: 202},
		{name: "move", a: 405, b: 539},
		{name: "move", a: 405, b: 539},
		{name: "button", a: int(buttonLeft), b: 0},
	}
	if len(be.calls) != len(want) {
		t.Fatalf("calls = %v", be.names())
	}
	for i, w := range want {
		if be.calls[i] != w {
			t.Fatalf("call %d = %+v, want %+v", i, be.calls[i], w)
		}
	}
	for _, c := range be.calls {
		if c.name == "releaseAll" {
			t.Fatal("a completed drag must not force-release buttons")
		}
	}
}
