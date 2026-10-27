package screen

import "errors"

// backend abstracts the host: display geometry, pointer position and input
// injection. Every platform provides one implementation.
type backend interface {
	// displays reports capturable desktops in host coordinates.
	displays() ([]Display, error)
	// pointerPos returns the current pointer position in host coordinates.
	pointerPos() (x, y int, ok bool)
	// movePointer warps the pointer to an absolute host position.
	movePointer(x, y int) error
	// button presses or releases a mouse button (buttonLeft/... values).
	button(b byte, down bool) error
	// wheel scrolls by lines; positive dy scrolls down.
	wheel(dx, dy int) error
	// key presses or releases the key identified by a DOM KeyboardEvent.code.
	key(code string, down bool) error
	// text types a UTF-8 string, using the host's unicode input path.
	text(s string) error
	// releaseAll lifts every button and modifier this backend may hold.
	releaseAll()
	// info reports whether input injection works right now.
	info() InputInfo
	// close releases host handles.
	close()
}

// unavailableBackend is used when the platform cannot inject input (missing
// permission, no X11 display, unsupported OS). It still lets the viewer *watch*
// when capture works, which is why it is a backend rather than an error.
type unavailableBackend struct {
	name   string
	reason string
}

func (b *unavailableBackend) displays() ([]Display, error) { return nil, errors.New(b.reason) }
func (b *unavailableBackend) pointerPos() (int, int, bool) { return 0, 0, false }
func (b *unavailableBackend) movePointer(int, int) error   { return errors.New(b.reason) }
func (b *unavailableBackend) button(byte, bool) error      { return errors.New(b.reason) }
func (b *unavailableBackend) wheel(int, int) error         { return errors.New(b.reason) }
func (b *unavailableBackend) key(string, bool) error       { return errors.New(b.reason) }
func (b *unavailableBackend) text(string) error            { return errors.New(b.reason) }
func (b *unavailableBackend) releaseAll()                  {}
func (b *unavailableBackend) close()                       {}
func (b *unavailableBackend) info() InputInfo {
	return inputUnavailable(b.reason, b.name)
}

// cursorShape is fixed for now: no supported platform exposes the system cursor
// shape cheaply, and an arrow is what the overlay needs 99% of the time.
func (m *Manager) cursorShape() byte { return shapeArrow }

func inputUnavailable(reason, backendName string) InputInfo {
	return InputInfo{Available: false, Reason: reason, Backend: backendName}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
