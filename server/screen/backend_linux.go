//go:build linux

package screen

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"github.com/jezek/xgb/xtest"
)

// x11Key is one resolved keycode plus the shift level it lives on.
type x11Key struct {
	code  byte
	shift bool
}

type x11Backend struct {
	mu      sync.Mutex
	conn    *xgb.Conn
	root    xproto.Window
	width   int
	height  int
	keymap  map[uint32]x11Key
	held    map[byte]bool
	heldMod map[string]bool
}

func newBackend() backend {
	if os.Getenv("DISPLAY") == "" {
		return &unavailableBackend{
			name:   "x11",
			reason: "DISPLAY is not set: input injection needs an X11 session",
		}
	}
	conn, err := xgb.NewConn()
	if err != nil {
		return &unavailableBackend{name: "x11", reason: "cannot open X display: " + err.Error()}
	}
	if err := xtest.Init(conn); err != nil {
		conn.Close()
		return &unavailableBackend{name: "x11", reason: "XTEST extension missing: " + err.Error()}
	}
	setup := xproto.Setup(conn)
	screen := setup.DefaultScreen(conn)
	b := &x11Backend{
		conn:    conn,
		root:    screen.Root,
		width:   int(screen.WidthInPixels),
		height:  int(screen.HeightInPixels),
		keymap:  make(map[uint32]x11Key),
		held:    make(map[byte]bool),
		heldMod: make(map[string]bool),
	}
	b.loadKeymap(setup.MinKeycode, setup.MaxKeycode)
	return b
}

// loadKeymap resolves every keysym the server knows about to a keycode. A
// missing keysym simply means "this layout cannot type that character".
func (b *x11Backend) loadKeymap(min, max xproto.Keycode) {
	count := byte(max-min) + 1
	reply, err := xproto.GetKeyboardMapping(b.conn, min, count).Reply()
	if err != nil {
		return
	}
	per := int(reply.KeysymsPerKeycode)
	if per == 0 {
		return
	}
	for i := 0; i < int(count) && i*per < len(reply.Keysyms); i++ {
		for level := 0; level < per && level < 4; level++ {
			ks := uint32(reply.Keysyms[i*per+level])
			if ks == 0 {
				continue
			}
			if _, exists := b.keymap[ks]; exists {
				continue
			}
			b.keymap[ks] = x11Key{code: byte(min) + byte(i), shift: level == 1}
		}
	}
}

func (b *x11Backend) displays() ([]Display, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	geom, err := xproto.GetGeometry(b.conn, xproto.Drawable(b.root)).Reply()
	if err != nil {
		return nil, err
	}
	return []Display{{
		ID:      "root",
		Name:    "X11 root window",
		W:       int(geom.Width),
		H:       int(geom.Height),
		Scale:   1,
		Primary: true,
	}}, nil
}

func (b *x11Backend) pointerPos() (int, int, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	reply, err := xproto.QueryPointer(b.conn, b.root).Reply()
	if err != nil {
		return 0, 0, false
	}
	return int(reply.RootX), int(reply.RootY), true
}

func (b *x11Backend) movePointer(x, y int) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return xproto.WarpPointer(b.conn, 0, b.root, 0, 0, 0, 0,
		int16(x), int16(y)).Check()
}

func (b *x11Backend) button(btn byte, down bool) error {
	typ := byte(xproto.ButtonPress)
	if !down {
		typ = xproto.ButtonRelease
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.held[btn] = down
	return xtest.FakeInput(b.conn, typ, btn, 0, b.root, 0, 0, 0).Check()
}

func (b *x11Backend) wheel(dx, dy int) error {
	steps := make([]byte, 0, abs(dy)+abs(dx))
	for i := 0; i < abs(dy); i++ {
		if dy > 0 {
			steps = append(steps, 5) // scroll down
		} else {
			steps = append(steps, 4) // scroll up
		}
	}
	for i := 0; i < abs(dx); i++ {
		if dx > 0 {
			steps = append(steps, 7) // scroll right
		} else {
			steps = append(steps, 6) // scroll left
		}
	}
	var lastErr error
	for _, btn := range steps {
		if err := b.button(btn, true); err != nil {
			lastErr = err
		}
		if err := b.button(btn, false); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (b *x11Backend) key(code string, down bool) error {
	ksym, ok := x11Keysyms[code]
	if !ok {
		return fmt.Errorf("unsupported key %q", code)
	}
	b.mu.Lock()
	key, ok := b.keymap[ksym]
	b.mu.Unlock()
	if !ok {
		return fmt.Errorf("key %q is not present in this keyboard layout", code)
	}
	if isModifierKey(code) {
		b.mu.Lock()
		b.heldMod[code] = down
		b.mu.Unlock()
	}
	return b.pressKeycode(key.code, down)
}

func (b *x11Backend) pressKeycode(code byte, down bool) error {
	typ := byte(xproto.KeyPress)
	if !down {
		typ = xproto.KeyRelease
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return xtest.FakeInput(b.conn, typ, code, 0, b.root, 0, 0, 0).Check()
}

func (b *x11Backend) pressKeysym(ksym uint32) error {
	b.mu.Lock()
	key, ok := b.keymap[ksym]
	b.mu.Unlock()
	if !ok {
		return fmt.Errorf("character U+%04X is not in this keyboard layout", ksym)
	}
	shift := x11Key{}
	if key.shift {
		shiftKey, found := b.keymap[shiftKeysym]
		if !found {
			return fmt.Errorf("shift key is not in this keyboard layout")
		}
		shift = shiftKey
		if err := b.pressKeycode(shift.code, true); err != nil {
			return err
		}
	}
	if err := b.pressKeycode(key.code, true); err != nil {
		return err
	}
	if err := b.pressKeycode(key.code, false); err != nil {
		return err
	}
	if key.shift {
		_ = b.pressKeycode(shift.code, false)
	}
	return nil
}

const shiftKeysym = 0xffe1

func (b *x11Backend) text(s string) error {
	for _, r := range s {
		ksym := uint32(r)
		if r >= 0x100 {
			ksym = 0x01000000 | uint32(r)
		}
		if err := b.pressKeysym(ksym); err != nil {
			return err
		}
		time.Sleep(4 * time.Millisecond)
	}
	return nil
}

func (b *x11Backend) releaseAll() {
	b.mu.Lock()
	buttons := make([]byte, 0, len(b.held))
	for btn, down := range b.held {
		if down {
			buttons = append(buttons, btn)
		}
	}
	mods := make([]string, 0, len(b.heldMod))
	for code, down := range b.heldMod {
		if down {
			mods = append(mods, code)
		}
	}
	b.held = make(map[byte]bool)
	b.heldMod = make(map[string]bool)
	b.mu.Unlock()

	for _, btn := range buttons {
		_ = b.button(btn, false)
	}
	for _, code := range mods {
		ksym, ok := x11Keysyms[code]
		if !ok {
			continue
		}
		b.mu.Lock()
		key, found := b.keymap[ksym]
		b.mu.Unlock()
		if found {
			_ = b.pressKeycode(key.code, false)
		}
	}
}

func (b *x11Backend) info() InputInfo {
	return InputInfo{Available: true, Permission: "granted", Backend: "x11-xtest"}
}

func (b *x11Backend) close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.conn != nil {
		b.conn.Close()
		b.conn = nil
	}
}

// promptPermission has nothing to prompt for on X11: XTEST either works or the
// session is not X11 at all.
func promptPermission(string) error { return errNotSupported }
