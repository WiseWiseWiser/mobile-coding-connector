package screen

import "testing"

func TestImageToHostPrimaryDisplay(t *testing.T) {
	d := Display{ID: "main", W: 1728, H: 1117, Scale: 2}
	cases := []struct {
		name   string
		ix, iy int
		wantX  int
		wantY  int
	}{
		{"origin", 0, 0, 0, 0},
		{"center", 640, 414, 864, 558},
		// Integer math floors: the last image pixel maps just short of the display edge.
		{"bottom right", 1279, 827, 1726, 1115},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			x, y := imageToHost(d, 1280, 828, tc.ix, tc.iy)
			if x != tc.wantX || y != tc.wantY {
				t.Fatalf("imageToHost = %d,%d want %d,%d", x, y, tc.wantX, tc.wantY)
			}
		})
	}
}

func TestImageToHostClampsOutOfRange(t *testing.T) {
	d := Display{W: 1728, H: 1117}
	if x, y := imageToHost(d, 1280, 828, -20, -20); x != 0 || y != 0 {
		t.Fatalf("negative coords = %d,%d", x, y)
	}
	if x, y := imageToHost(d, 1280, 828, 99999, 99999); x != 1726 || y != 1115 {
		t.Fatalf("oversized coords = %d,%d", x, y)
	}
}

func TestImageToHostSecondDisplayOffset(t *testing.T) {
	d := Display{ID: "2", X: 1440, Y: 120, W: 1920, H: 1080}
	x, y := imageToHost(d, 1440, 810, 0, 0)
	if x != 1440 || y != 120 {
		t.Fatalf("origin = %d,%d want 1440,120", x, y)
	}
	x, y = imageToHost(d, 1440, 810, 720, 405)
	if x != 2400 || y != 660 {
		t.Fatalf("center = %d,%d want 2400,660", x, y)
	}
}

func TestHostToImage(t *testing.T) {
	d := Display{W: 1728, H: 1117}
	if ix, iy := hostToImage(d, 1280, 828, 864, 558); ix != 640 || iy != 413 {
		t.Fatalf("hostToImage = %d,%d want 640,413", ix, iy)
	}
	// Coordinates outside the display must clamp into the frame.
	if ix, iy := hostToImage(d, 1280, 828, 5000, 5000); ix != 1279 || iy != 827 {
		t.Fatalf("clamped = %d,%d", ix, iy)
	}
}

func TestMappingRoundTrip(t *testing.T) {
	d := Display{X: 100, Y: 40, W: 1920, H: 1080}
	for _, p := range [][2]int{{0, 0}, {1, 1}, {100, 200}, {640, 360}, {1279, 719}} {
		hx, hy := imageToHost(d, 1280, 720, p[0], p[1])
		bx, by := hostToImage(d, 1280, 720, hx, hy)
		if diff := bx - p[0]; diff < -1 || diff > 1 {
			t.Fatalf("round trip x: %d -> %d", p[0], bx)
		}
		if diff := by - p[1]; diff < -1 || diff > 1 {
			t.Fatalf("round trip y: %d -> %d", p[1], by)
		}
	}
}

func TestMappingDegenerateInput(t *testing.T) {
	if x, y := imageToHost(Display{}, 0, 0, 5, 5); x != 0 || y != 0 {
		t.Fatalf("empty display = %d,%d", x, y)
	}
	if x, y := hostToImage(Display{W: 10, H: 10}, 0, 10, 5, 5); x != 0 || y != 0 {
		t.Fatalf("empty image = %d,%d", x, y)
	}
}

func TestClampInt(t *testing.T) {
	if clampInt(5, 0, 3) != 3 || clampInt(-5, 0, 3) != 0 || clampInt(2, 0, 3) != 2 {
		t.Fatal("clampInt misbehaves")
	}
}
