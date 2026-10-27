//go:build !darwin

package screen

import "context"

// startPlatformSource captures with ffmpeg everywhere except macOS, where
// ScreenCaptureKit is required on macOS 26+.
func startPlatformSource(ctx context.Context, maxWidth, fps, quality int) (source, error) {
	args, err := captureArgs(maxWidth, fps)
	if err != nil {
		return nil, err
	}
	return startFFmpeg(ctx, args)
}
