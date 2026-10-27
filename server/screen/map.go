package screen

// hostToImage converts host coordinates (points on macOS, pixels on X11) to
// pixel coordinates inside the streamed frame.
func hostToImage(d Display, imgW, imgH, x, y int) (int, int) {
	if d.W <= 0 || d.H <= 0 || imgW <= 0 || imgH <= 0 {
		return 0, 0
	}
	ix := (x - d.X) * imgW / d.W
	iy := (y - d.Y) * imgH / d.H
	return clampInt(ix, 0, imgW-1), clampInt(iy, 0, imgH-1)
}

// imageToHost converts frame pixel coordinates back to host coordinates, which
// is what input injection consumes.
func imageToHost(d Display, imgW, imgH, ix, iy int) (int, int) {
	if d.W <= 0 || d.H <= 0 || imgW <= 0 || imgH <= 0 {
		return 0, 0
	}
	ix = clampInt(ix, 0, imgW-1)
	iy = clampInt(iy, 0, imgH-1)
	x := d.X + ix*d.W/imgW
	y := d.Y + iy*d.H/imgH
	return clampInt(x, d.X, d.X+d.W-1), clampInt(y, d.Y, d.Y+d.H-1)
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
