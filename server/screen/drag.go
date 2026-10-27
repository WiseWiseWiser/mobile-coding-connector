package screen

// dragButtonFor reports which mouse button is holding the pointer down, if any.
//
// A pointer move posted while a button is down must be a "dragged" event:
// Cocoa delivers kCGEventMouseMoved as a hover, so an app never starts a drag
// even though the button went down and the pointer travelled. The backend asks
// this before posting every move.
//
// Left wins over right over middle — a pointer can only drag with one button,
// and the left button is the one a drag gesture actually uses.
func dragButtonFor(held map[byte]bool) byte {
	switch {
	case held[buttonLeft]:
		return buttonLeft
	case held[buttonRight]:
		return buttonRight
	case held[buttonMiddle]:
		return buttonMiddle
	default:
		return buttonNone
	}
}
