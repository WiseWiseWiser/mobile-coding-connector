package screen

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

func newTestImage(w, h int, fill color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, fill)
		}
	}
	return img
}

func TestTilerFirstFrameIsFull(t *testing.T) {
	img := newTestImage(320, 200, color.RGBA{10, 20, 30, 255})
	tl := newTiler(TileSize, 50)
	tiles, full := tl.diff(img)
	if !full {
		t.Fatal("first frame should be full")
	}
	if len(tiles) != 3*2 {
		t.Fatalf("expected 6 tiles, got %d", len(tiles))
	}
}

func TestTilerSkipsUnchangedTiles(t *testing.T) {
	img := newTestImage(320, 200, color.RGBA{10, 20, 30, 255})
	tl := newTiler(TileSize, 50)
	tl.diff(img)

	tiles, full := tl.diff(img)
	if len(tiles) != 0 {
		t.Fatalf("expected no tiles for identical frame, got %d", len(tiles))
	}
	if full {
		t.Fatal("identical frame must not be full")
	}
}

// ScreenCaptureKit re-composites each frame, so an idle desktop still arrives
// with every channel off by a few units. That noise must not mark tiles dirty,
// otherwise every frame repaints the whole screen.
func TestTilerIgnoresCaptureNoise(t *testing.T) {
	img := newTestImage(320, 200, color.RGBA{100, 120, 140, 255})
	tl := newTiler(TileSize, 50)
	tl.diff(img)

	for i := 1; i <= 5; i++ {
		off := uint8(i)
		for y := 0; y < 200; y++ {
			for x := 0; x < 320; x++ {
				img.SetRGBA(x, y, color.RGBA{100 + off, 120 - off, 140 + off, 255})
			}
		}
		if tiles, full := tl.diff(img); len(tiles) != 0 || full {
			t.Fatalf("frame %d: noise produced %d tiles (full=%v)", i, len(tiles), full)
		}
	}
}

func TestTilerStillSeesRealChangeThroughNoise(t *testing.T) {
	img := newTestImage(320, 200, color.RGBA{100, 120, 140, 255})
	tl := newTiler(TileSize, 50)
	tl.diff(img)

	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			img.SetRGBA(x, y, color.RGBA{103, 122, 143, 255})
		}
	}
	img.SetRGBA(200, 70, color.RGBA{250, 250, 250, 255})

	tiles, _ := tl.diff(img)
	if len(tiles) != 1 {
		t.Fatalf("expected 1 changed tile, got %d", len(tiles))
	}
	if tiles[0].X != 128 || tiles[0].Y != 0 {
		t.Fatalf("unexpected tile origin: %d,%d", tiles[0].X, tiles[0].Y)
	}
}

func TestTilerReportsOnlyChangedTile(t *testing.T) {
	img := newTestImage(320, 200, color.RGBA{10, 20, 30, 255})
	tl := newTiler(TileSize, 50)
	tl.diff(img)

	img.SetRGBA(150, 130, color.RGBA{255, 255, 255, 255})

	tiles, _ := tl.diff(img)
	if len(tiles) != 1 {
		t.Fatalf("expected 1 changed tile, got %d", len(tiles))
	}
	if tiles[0].X != 128 || tiles[0].Y != 128 {
		t.Fatalf("unexpected tile origin: %d,%d", tiles[0].X, tiles[0].Y)
	}
}

func TestTileJPEGDecodes(t *testing.T) {
	img := newTestImage(300, 140, color.RGBA{1, 2, 3, 255})
	tl := newTiler(TileSize, 60)
	tiles, _ := tl.diff(img)
	if len(tiles) != 3*2 {
		t.Fatalf("expected 6 tiles, got %d", len(tiles))
	}
	last := tiles[len(tiles)-1]
	src, err := jpeg.Decode(bytes.NewReader(last.JPEG))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	// The final column/row is partial: 300-256=44 wide, 140-128=12 high.
	if got := src.Bounds().Dx(); got != 44 {
		t.Fatalf("last tile width = %d, want 44", got)
	}
	if got := src.Bounds().Dy(); got != 12 {
		t.Fatalf("last tile height = %d, want 12", got)
	}
}

func TestTilerResetsOnGridChange(t *testing.T) {
	tl := newTiler(TileSize, 50)
	tl.diff(newTestImage(320, 200, color.RGBA{1, 1, 1, 255}))
	tiles, full := tl.diff(newTestImage(640, 400, color.RGBA{1, 1, 1, 255}))
	if !full {
		t.Fatal("grid change must produce a full frame")
	}
	if len(tiles) != 5*4 {
		t.Fatalf("expected 20 tiles, got %d", len(tiles))
	}
}
