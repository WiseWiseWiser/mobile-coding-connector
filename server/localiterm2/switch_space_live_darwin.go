//go:build darwin

package localiterm2

import (
	"fmt"

	spacelib "github.com/xhd2015/dot-pkgs/go-pkgs/computer-use/macos/space"
)

// liveSwitchSpaceID changes the *visible* Desktop via Mission Control
// (open Spaces Bar, click "Desktop N"). That is the Hammerspoon hs.spaces
// path. SLSManagedDisplaySetCurrentSpace only mutates WindowServer on
// macOS 15+ and does not repaint, so other Spaces' windows appear on
// the frozen Desktop.
func liveSwitchSpaceID(spaceID uint64) error {
	users, err := spacelib.ListUserSpaces()
	if err != nil {
		return fmt.Errorf("switch space: list: %w", err)
	}
	for _, u := range users {
		if u.ID != spaceID {
			continue
		}
		if u.Current {
			return nil
		}
		n := u.Index + 1
		if err := spacelib.Switch(n, nil); err != nil {
			return fmt.Errorf("switch space: %w", err)
		}
		return nil
	}
	return fmt.Errorf("switch space: space_id %d not found", spaceID)
}
