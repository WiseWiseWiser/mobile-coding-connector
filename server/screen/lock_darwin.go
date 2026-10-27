//go:build darwin && cgo

package screen

/*
#include <CoreGraphics/CoreGraphics.h>

// ac_screen_locked reports whether the login window currently owns the display.
// While it does, injected events would go to the unlock prompt, so input has to
// stop: a remote viewer must never be able to type into a password field.
static int ac_screen_locked(void) {
	CFDictionaryRef session = CGSessionCopyCurrentDictionary();
	if (session == NULL) {
		return 0;
	}
	int locked = 0;
	CFBooleanRef flag = (CFBooleanRef)CFDictionaryGetValue(session, CFSTR("CGSSessionScreenIsLocked"));
	if (flag != NULL && CFGetTypeID(flag) == CFBooleanGetTypeID() && CFBooleanGetValue(flag)) {
		locked = 1;
	}
	CFRelease(session);
	return locked;
}
*/
import "C"

import (
	"sync"
	"time"
)

// The lock state changes rarely; caching keeps the input hot path off CoreGraphics.
const lockStateTTL = 500 * time.Millisecond

var lockState struct {
	mu     sync.Mutex
	at     time.Time
	locked bool
}

func screenLocked() bool {
	lockState.mu.Lock()
	defer lockState.mu.Unlock()
	if time.Since(lockState.at) < lockStateTTL {
		return lockState.locked
	}
	lockState.locked = C.ac_screen_locked() != 0
	lockState.at = time.Now()
	return lockState.locked
}
