//go:build darwin && cgo

package screen

/*
#cgo LDFLAGS: -framework ApplicationServices -framework CoreGraphics
#include <ApplicationServices/ApplicationServices.h>
#include <CoreGraphics/CoreGraphics.h>
#include <stdlib.h>

static int ac_ax_trusted(void) {
	return AXIsProcessTrusted() ? 1 : 0;
}

static int ac_post_event_access(void) {
	if (__builtin_available(macOS 10.15, *)) {
		return CGPreflightPostEventAccess() ? 1 : 0;
	}
	return AXIsProcessTrusted() ? 1 : 0;
}

static int ac_request_post_event_access(void) {
	if (__builtin_available(macOS 10.15, *)) {
		return CGRequestPostEventAccess() ? 1 : 0;
	}
	return 0;
}

static int ac_request_screen_capture(void) {
	if (__builtin_available(macOS 10.15, *)) {
		return CGRequestScreenCaptureAccess() ? 1 : 0;
	}
	return 0;
}

typedef struct {
	double x, y, w, h;
	double pixel_w, pixel_h;
} ac_display;

static int ac_main_display(ac_display *out) {
	CGDirectDisplayID did = CGMainDisplayID();
	CGRect b = CGDisplayBounds(did);
	if (b.size.width <= 0 || b.size.height <= 0) {
		return 0;
	}
	out->x = b.origin.x;
	out->y = b.origin.y;
	out->w = b.size.width;
	out->h = b.size.height;
	out->pixel_w = (double)CGDisplayPixelsWide(did);
	out->pixel_h = (double)CGDisplayPixelsHigh(did);
	return 1;
}

static void ac_pointer(double *x, double *y) {
	CGEventRef ev = CGEventCreate(NULL);
	if (ev == NULL) {
		*x = 0;
		*y = 0;
		return;
	}
	CGPoint p = CGEventGetLocation(ev);
	*x = p.x;
	*y = p.y;
	CFRelease(ev);
}

// ac_move posts a pointer move. When a button is held (button != 0) it has to be
// a dragged event: Cocoa apps treat every kCGEventMouseMoved as a hover, so a
// pressed button would follow the pointer without dragging anything.
static void ac_move(double x, double y, int button) {
	CGEventType type = kCGEventMouseMoved;
	CGMouseButton mb = kCGMouseButtonLeft;
	switch (button) {
	case 1:
		type = kCGEventLeftMouseDragged;
		mb = kCGMouseButtonLeft;
		break;
	case 2:
		type = kCGEventOtherMouseDragged;
		mb = kCGMouseButtonCenter;
		break;
	case 3:
		type = kCGEventRightMouseDragged;
		mb = kCGMouseButtonRight;
		break;
	}
	CGEventRef ev = CGEventCreateMouseEvent(NULL, type, CGPointMake(x, y), mb);
	if (ev == NULL) {
		return;
	}
	CGEventPost(kCGHIDEventTap, ev);
	CFRelease(ev);
}

static void ac_button(int button, int down, int clicks, double x, double y) {
	CGEventType type;
	CGMouseButton mb;
	switch (button) {
	case 2:
		type = down ? kCGEventOtherMouseDown : kCGEventOtherMouseUp;
		mb = kCGMouseButtonCenter;
		break;
	case 3:
		type = down ? kCGEventRightMouseDown : kCGEventRightMouseUp;
		mb = kCGMouseButtonRight;
		break;
	default:
		type = down ? kCGEventLeftMouseDown : kCGEventLeftMouseUp;
		mb = kCGMouseButtonLeft;
		break;
	}
	CGEventRef ev = CGEventCreateMouseEvent(NULL, type, CGPointMake(x, y), mb);
	if (ev == NULL) {
		return;
	}
	if (clicks > 1) {
		CGEventSetIntegerValueField(ev, kCGMouseEventClickState, (int64_t)clicks);
	}
	CGEventPost(kCGHIDEventTap, ev);
	CFRelease(ev);
}

static void ac_scroll(int lines, double x, double y) {
	CGEventRef ev = CGEventCreateScrollWheelEvent(NULL, kCGScrollEventUnitLine, 1, (int32_t)(-lines));
	if (ev == NULL) {
		return;
	}
	CGEventSetLocation(ev, CGPointMake(x, y));
	CGEventPost(kCGHIDEventTap, ev);
	CFRelease(ev);
}

static void ac_key(unsigned short code, int down) {
	CGEventRef ev = CGEventCreateKeyboardEvent(NULL, (CGKeyCode)code, down != 0);
	if (ev == NULL) {
		return;
	}
	CGEventPost(kCGHIDEventTap, ev);
	CFRelease(ev);
}

// ac_text posts one unicode string as a key with a synthetic keycode, which is
// how non-US layouts and emoji reach the focused app.
static void ac_text(const UniChar *chars, int len) {
	CGEventRef down = CGEventCreateKeyboardEvent(NULL, 0, true);
	if (down != NULL) {
		CGEventKeyboardSetUnicodeString(down, (UniCharCount)len, chars);
		CGEventPost(kCGHIDEventTap, down);
		CFRelease(down);
	}
	CGEventRef up = CGEventCreateKeyboardEvent(NULL, 0, false);
	if (up != NULL) {
		CGEventKeyboardSetUnicodeString(up, (UniCharCount)len, chars);
		CGEventPost(kCGHIDEventTap, up);
		CFRelease(up);
	}
}
*/
import "C"

import (
	"errors"
	"fmt"
	"os/exec"
	"sync"
	"time"
	"unicode/utf16"
	"unsafe"
)

type darwinBackend struct {
	mu    sync.Mutex
	held  map[byte]bool
	mods  map[string]bool
	click clickTracker
}

func newBackend() backend {
	return &darwinBackend{
		held: make(map[byte]bool),
		mods: make(map[string]bool),
	}
}

func (b *darwinBackend) displays() ([]Display, error) {
	var d C.ac_display
	if C.ac_main_display(&d) == 0 {
		return nil, errors.New("no active display reported by CoreGraphics")
	}
	scale := 1.0
	if d.w > 0 {
		scale = float64(d.pixel_w) / float64(d.w)
	}
	if scale <= 0 {
		scale = 1
	}
	return []Display{{
		ID:      "main",
		Name:    "Main display",
		X:       int(d.x),
		Y:       int(d.y),
		W:       int(d.w),
		H:       int(d.h),
		Scale:   scale,
		Primary: true,
	}}, nil
}

func (b *darwinBackend) pointerPos() (int, int, bool) {
	var x, y C.double
	C.ac_pointer(&x, &y)
	return int(x), int(y), true
}

func (b *darwinBackend) movePointer(x, y int) error {
	C.ac_move(C.double(x), C.double(y), C.int(b.dragButton()))
	return nil
}

// dragButton reports the button the pointer is currently held down with, so the
// move can be posted as a drag instead of a hover.
func (b *darwinBackend) dragButton() byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return dragButtonFor(b.held)
}

func (b *darwinBackend) button(btn byte, down bool) error {
	b.mu.Lock()
	x, y, _ := b.pointerPos()
	clicks := b.click.click(btn, down, x, y)
	b.held[btn] = down
	b.mu.Unlock()
	C.ac_button(C.int(btn), boolToC(down), C.int(clicks), C.double(x), C.double(y))
	return nil
}

func (b *darwinBackend) wheel(dx, dy int) error {
	x, y, _ := b.pointerPos()
	if dy != 0 {
		C.ac_scroll(C.int(dy), C.double(x), C.double(y))
	}
	for i := 0; i < abs(dx); i++ {
		lines := 1
		if dx < 0 {
			lines = -1
		}
		C.ac_scroll(C.int(lines), C.double(x), C.double(y))
	}
	return nil
}

func (b *darwinBackend) key(code string, down bool) error {
	kc, ok := darwinKeyCodes[code]
	if !ok {
		return fmt.Errorf("unsupported key %q", code)
	}
	if isModifierKey(code) {
		b.mu.Lock()
		b.mods[code] = down
		b.mu.Unlock()
	}
	C.ac_key(C.ushort(kc), boolToC(down))
	return nil
}

func (b *darwinBackend) text(s string) error {
	const chunk = 16
	runes := []rune(s)
	for len(runes) > 0 {
		n := min(chunk, len(runes))
		units := utf16.Encode(runes[:n])
		if len(units) > 0 {
			C.ac_text((*C.UniChar)(unsafe.Pointer(&units[0])), C.int(len(units)))
		}
		runes = runes[n:]
		time.Sleep(8 * time.Millisecond)
	}
	return nil
}

func (b *darwinBackend) releaseAll() {
	b.mu.Lock()
	buttons := make([]byte, 0, len(b.held))
	for btn, down := range b.held {
		if down {
			buttons = append(buttons, btn)
		}
	}
	mods := make([]string, 0, len(b.mods))
	for code, down := range b.mods {
		if down {
			mods = append(mods, code)
		}
	}
	b.held = make(map[byte]bool)
	b.mods = make(map[string]bool)
	b.mu.Unlock()

	// Post the button-ups where the pointer actually is: a mouse event carries a
	// location, so the old (0,0) teleported the host cursor into the top-left
	// corner every time control was dropped mid-drag.
	x, y, _ := b.pointerPos()
	for _, btn := range buttons {
		C.ac_button(C.int(btn), 0, 0, C.double(x), C.double(y))
	}
	for _, code := range mods {
		if kc, ok := darwinKeyCodes[code]; ok {
			C.ac_key(C.ushort(kc), 0)
		}
	}
}

func (b *darwinBackend) info() InputInfo {
	info := InputInfo{Backend: "cgevent"}
	trusted := C.ac_ax_trusted() != 0
	postAccess := C.ac_post_event_access() != 0
	info.AXTrusted = trusted
	info.PostEventAccess = postAccess
	switch {
	case postAccess || trusted:
		info.Available = true
		info.Permission = "granted"
	case true:
		info.Available = false
		info.Permission = "denied"
		info.CanPrompt = true
		info.Reason = "Accessibility permission is required: System Settings > Privacy & Security > Accessibility, then enable the ai-critic binary (or the terminal running it)"
	}
	return info
}

func (b *darwinBackend) close() {}

// ---------------------------------------------------------------- permissions

func promptPermission(which string) error {
	switch which {
	case "accessibility", "input", "":
		C.ac_request_post_event_access()
		return openSystemSettings("x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility")
	case "screen", "screen-recording":
		C.ac_request_screen_capture()
		return openSystemSettings("x-apple.systempreferences:com.apple.preference.security?Privacy_ScreenCapture")
	}
	return fmt.Errorf("unknown permission %q", which)
}

func openSystemSettings(target string) error {
	return exec.Command("open", target).Run()
}

func boolToC(v bool) C.int {
	if v {
		return 1
	}
	return 0
}

// clickTracker reconstructs the click count macOS needs for double/triple click
// to be recognized: each event carries how many clicks it belongs to.
type clickTracker struct {
	last   time.Time
	button byte
	x, y   int
	count  int
}

func (c *clickTracker) click(btn byte, down bool, x, y int) int {
	if !down {
		return c.count
	}
	now := time.Now()
	if c.button == btn && now.Sub(c.last) < 500*time.Millisecond &&
		abs(x-c.x) <= 4 && abs(y-c.y) <= 4 {
		c.count++
	} else {
		c.count = 1
	}
	if c.count > 3 {
		c.count = 1
	}
	c.last = now
	c.button = btn
	c.x, c.y = x, y
	return c.count
}
