package screen

import (
	"bytes"
	"image"
	"image/jpeg"
	"math"
)

// tileTolerance is how far a single channel may drift before the tile counts as
// changed. ScreenCaptureKit re-composites every frame, so pixel values wobble by
// a few units even when the desktop stands still; comparing exactly turned that
// noise into a full-screen repaint on every single frame.
const tileTolerance = 8

// tiler splits consecutive frames into fixed-size tiles and reports only the
// ones whose pixels changed. Comparing source pixels (not the encoded JPEG)
// keeps the decision deterministic and lets us skip the expensive encode for
// untouched regions.
type tiler struct {
	tileSize int
	quality  int

	width  int
	height int
	cols   int
	rows   int

	// prev is the last published image: the pixels viewers currently hold. It
	// doubles as the keyframe a late viewer is sent.
	prev   *image.RGBA
	primed bool
}

func newTiler(tileSize, quality int) *tiler {
	return &tiler{tileSize: tileSize, quality: quality}
}

// reset forgets the previous frame so the next diff yields every tile.
func (t *tiler) reset() {
	t.primed = false
}

// diff returns the changed tiles for img. full reports that the caller should
// treat the frame as a keyframe (first frame after reset, grid change, or most
// of the screen changed anyway).
func (t *tiler) diff(img *image.RGBA) (tiles []Tile, full bool) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if t.tileSize <= 0 || w <= 0 || h <= 0 {
		return nil, false
	}
	tile := t.tileSize
	cols := (w + tile - 1) / tile
	rows := (h + tile - 1) / tile
	if cols != t.cols || rows != t.rows || w != t.width || h != t.height {
		t.width, t.height = w, h
		t.cols, t.rows = cols, rows
		t.prev = nil
		t.primed = false
	}

	firstFrame := !t.primed || t.prev == nil
	if firstFrame {
		t.prev = image.NewRGBA(image.Rect(0, 0, w, h))
	}
	tiles = make([]Tile, 0, 8)
	for ty := 0; ty < rows; ty++ {
		for tx := 0; tx < cols; tx++ {
			x0 := tx * tile
			y0 := ty * tile
			rect := image.Rect(x0, y0, min(x0+tile, w), min(y0+tile, h))
			if !firstFrame && !tileChanged(t.prev, img, rect, tileTolerance) {
				continue
			}
			data, ok := encodeTile(img, rect, t.quality)
			if !ok {
				continue
			}
			copyTile(t.prev, img, rect)
			tiles = append(tiles, Tile{X: x0, Y: y0, JPEG: data})
		}
	}
	t.primed = true
	// A frame that replaces most of the screen is cheaper for the client to
	// treat as a full repaint (one drawImage per tile anyway, but this also
	// signals "everything is here" so the client can drop stale content).
	full = firstFrame || len(tiles) >= int(math.Ceil(float64(cols*rows)*0.75))
	return tiles, full
}

// snapshot returns every tile of the last published frame (the layout a viewer
// joining a running stream has missed). Without it a late viewer keeps an empty
// canvas: nothing changes, no frame is published, and the keyframe it asked for
// never comes.
func (t *tiler) snapshot() []Tile {
	if t == nil || t.prev == nil || t.tileSize <= 0 {
		return nil
	}
	img := t.prev
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	tiles := make([]Tile, 0, t.cols*t.rows)
	for ty := 0; ty < t.rows; ty++ {
		for tx := 0; tx < t.cols; tx++ {
			rect := image.Rect(tx*t.tileSize, ty*t.tileSize, min((tx+1)*t.tileSize, w), min((ty+1)*t.tileSize, h))
			data, ok := encodeTile(img, rect, t.quality)
			if !ok {
				continue
			}
			tiles = append(tiles, Tile{X: rect.Min.X, Y: rect.Min.Y, JPEG: data})
		}
	}
	return tiles
}

// tileChanged reports whether any channel of the tile drifted further than tol.
// Rows that are byte-identical are skipped whole, which is the common case even
// for a tile that does contain a change.
func tileChanged(prev, img *image.RGBA, r image.Rectangle, tol int) bool {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		a := prev.Pix[prev.PixOffset(r.Min.X, y) : prev.PixOffset(r.Min.X, y)+r.Dx()*4]
		b := img.Pix[img.PixOffset(r.Min.X, y) : img.PixOffset(r.Min.X, y)+r.Dx()*4]
		if bytes.Equal(a, b) {
			continue
		}
		for i := range a {
			if d := int(a[i]) - int(b[i]); d > tol || d < -tol {
				return true
			}
		}
	}
	return false
}

func copyTile(dst, src *image.RGBA, r image.Rectangle) {
	rowBytes := r.Dx() * 4
	for y := r.Min.Y; y < r.Max.Y; y++ {
		copy(dst.Pix[dst.PixOffset(r.Min.X, y):dst.PixOffset(r.Min.X, y)+rowBytes],
			src.Pix[src.PixOffset(r.Min.X, y):src.PixOffset(r.Min.X, y)+rowBytes])
	}
}

func encodeTile(img *image.RGBA, rect image.Rectangle, quality int) ([]byte, bool) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img.SubImage(rect), &jpeg.Options{Quality: quality}); err != nil {
		return nil, false
	}
	data := make([]byte, buf.Len())
	copy(data, buf.Bytes())
	return data, true
}
