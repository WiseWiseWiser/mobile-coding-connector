//go:build !darwin

package localiterm2

import "fmt"

func liveSwitchSpaceID(spaceID uint64) error {
	return fmt.Errorf("switch space %d: unsupported platform", spaceID)
}
