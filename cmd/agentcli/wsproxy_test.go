package agentcli

import (
	"strings"
	"testing"

	"github.com/skip2/go-qrcode"
)

func TestCurrentQROutputShape(t *testing.T) {
	bmp := make([][]bool, 16)
	for i := range bmp {
		bmp[i] = make([]bool, 16)
		for j := 0; j < 8; j++ {
			bmp[i][j] = true
		}
	}

	out := renderQuadrantQRFromBitmap(bmp)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	outWidth := len([]rune(lines[0]))
	outHeight := len(lines)

	// 16x16 bitmap, 4-module quiet zone crop → 8x8 modules
	// Quadrant: (8+1)/2 = 4 wide, (8+1)/2 = 4 tall
	expectWidth := 4
	expectHeight := 4

	t.Logf("=== Quadrant QR with quiet zone crop ===")
	t.Logf("Mock bitmap:      16x16 (includes 4-module quiet zone)")
	t.Logf("After crop:       8x8 modules")
	t.Logf("Quadrant output:  %d chars wide x %d lines", outWidth, outHeight)

	if outWidth != expectWidth {
		t.Errorf("width = %d, want %d", outWidth, expectWidth)
	}
	if outHeight != expectHeight {
		t.Errorf("height = %d, want %d", outHeight, expectHeight)
	}
}

func TestRenderQuadrantQR_Dimensions(t *testing.T) {
	qr, err := qrcode.New("vmess://test.example.com:443", qrcode.Low)
	if err != nil {
		t.Fatalf("qrcode.New() error = %v", err)
	}

	bmp := qr.Bitmap()
	bmSize := len(bmp)
	moduleSize := bmSize - 8 // 4 quiet zone on each side

	out := renderQuadrantQR(qr)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	if len(lines) == 0 {
		t.Fatal("renderQuadrantQR returned no lines")
	}

	newWidth := len([]rune(lines[0]))
	newHeight := len(lines)

	if newWidth == 0 || newHeight == 0 {
		t.Fatal("output dimensions are zero")
	}

	expectWidth := (moduleSize + 1) / 2
	expectHeight := (moduleSize + 1) / 2

	if newWidth != expectWidth {
		t.Errorf("width = %d, want %d (bitmap size=%d, modules=%d)", newWidth, expectWidth, bmSize, moduleSize)
	}
	if newHeight != expectHeight {
		t.Errorf("height = %d, want %d (bitmap size=%d, modules=%d)", newHeight, expectHeight, bmSize, moduleSize)
	}

	if len(lines) > 1 {
		for _, line := range lines[1:] {
			if len([]rune(line)) != newWidth {
				t.Errorf("inconsistent line width: first line=%d, got=%d", newWidth, len([]rune(line)))
			}
		}
	}
}

func TestRenderQuadrantQR_ValidChars(t *testing.T) {
	qr, err := qrcode.New("hello-world", qrcode.Low)
	if err != nil {
		t.Fatalf("qrcode.New() error = %v", err)
	}

	out := renderQuadrantQR(qr)
	if out == "" {
		t.Fatal("renderQuadrantQR returned empty")
	}

	validChars := map[rune]bool{}
	for _, r := range quadrantChars {
		validChars[r] = true
	}

	for i, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		for j, r := range line {
			if r == '\n' || r == '\r' {
				continue
			}
			if !validChars[r] {
				t.Errorf("invalid char %q (U+%04X) at line %d col %d", string(r), r, i, j)
			}
		}
	}
}

func TestRenderQuadrantQR_NonEmpty(t *testing.T) {
	qr, err := qrcode.New("test", qrcode.Low)
	if err != nil {
		t.Fatalf("qrcode.New() error = %v", err)
	}

	out := renderQuadrantQR(qr)
	if strings.TrimSpace(out) == "" {
		t.Fatal("renderQuadrantQR returned only whitespace")
	}
}

func TestRenderQuadrantQR_KnownBitmap(t *testing.T) {
	bmp := [][]bool{
		{false, true, false},
		{true, true, false},
		{false, false, true},
	}

	out := renderQuadrantQRFromBitmap(bmp)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	// 3x3 bitmap, too small for quiet zone crop (quiet=4, 3<=8)
	// So quietZone becomes 0, full 3x3 rendered
	// Quadrant: ceil(3/2)=2 wide, ceil(3/2)=2 tall
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines for 3x3 bitmap, got %d", len(lines))
	}
	line0 := []rune(lines[0])
	line1 := []rune(lines[1])

	verifyQuadrantChar(t, line0[0], bmp[0][0], bmp[0][1], bmp[1][0], bmp[1][1])
	verifyQuadrantChar(t, line0[1], bmp[0][2], false, bmp[1][2], false)
	verifyQuadrantChar(t, line1[0], bmp[2][0], bmp[2][1], false, false)
	verifyQuadrantChar(t, line1[1], bmp[2][2], false, false, false)
}

func TestRenderQuadrantQR_QuietZoneCrop(t *testing.T) {
	// 10x10 bitmap simulating a QR with 4-module quiet zone
	// The inner 2x2 represents the actual QR modules
	bmp := make([][]bool, 10)
	for i := range bmp {
		bmp[i] = make([]bool, 10)
	}
	bmp[4][4] = true
	bmp[4][5] = true
	bmp[5][4] = false
	bmp[5][5] = true

	out := renderQuadrantQRFromBitmap(bmp)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	// After crop: 2x2 modules → 1 line × 1 char
	if len(lines) != 1 {
		t.Fatalf("expected 1 line after quiet zone crop, got %d", len(lines))
	}
	runes := []rune(lines[0])
	if len(runes) != 1 {
		t.Fatalf("expected 1 char, got %d", len(runes))
	}

	expected := quadrantChars[0b1101] // ul=true, ur=true, ll=false, lr=true → ▜
	if runes[0] != expected {
		t.Errorf("got %c (U+%04X), want %c (U+%04X)", runes[0], runes[0], expected, expected)
	}
}

func TestRenderQuadrantQR_EmptyBitmap(t *testing.T) {
	out := renderQuadrantQRFromBitmap([][]bool{})
	if out != "" {
		t.Errorf("expected empty string for empty bitmap, got %q", out)
	}
}

func TestRenderQuadrantQR_SmallBitmap(t *testing.T) {
	tests := []struct {
		name     string
		bmp      [][]bool
		expected rune
	}{
		{"1x1 all false", [][]bool{{false}}, ' '},
		{"1x1 all true", [][]bool{{true}}, '▘'},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := renderQuadrantQRFromBitmap(tt.bmp)
			out = strings.TrimRight(out, "\n")
			runes := []rune(out)
			if len(runes) != 1 {
				t.Fatalf("expected 1 char, got %q", out)
			}
			got := runes[0]
			if got != tt.expected {
				t.Errorf("got %c (U+%04X), want %c (U+%04X)", got, got, tt.expected, tt.expected)
			}
		})
	}
}

func TestQuadrantChars_AllDistinctOrSpace(t *testing.T) {
	seen := map[rune]int{}
	for i, r := range quadrantChars {
		if prevIdx, ok := seen[r]; ok {
			if r != ' ' {
				t.Errorf("quadrantChars[%d] and [%d] both map to %q (U+%04X)", prevIdx, i, string(r), r)
			}
		}
		seen[r] = i
	}
}

func TestGenerateQRCode_DefaultFullSize(t *testing.T) {
	out := generateQRCode("vmess://some-link", false)
	if out == "" {
		t.Fatal("empty output for valid content")
	}
	if strings.Contains(out, "error") {
		t.Fatal("unexpected error in output")
	}

	qr, _ := qrcode.New("vmess://some-link", qrcode.Low)
	expected := qr.ToSmallString(false)
	if out != expected {
		t.Errorf("default generateQRCode should match ToSmallString")
	}
}

func TestGenerateQRCode_SmallerSize(t *testing.T) {
	out := generateQRCode("vmess://some-link", true)
	if out == "" {
		t.Fatal("empty output for valid content")
	}
	if strings.Contains(out, "error") {
		t.Fatal("unexpected error in output")
	}

	qr, _ := qrcode.New("vmess://some-link", qrcode.Low)
	expected := renderQuadrantQR(qr)
	if out != expected {
		t.Errorf("smaller generateQRCode should match renderQuadrantQR")
	}
}

func TestGenerateQRCode_SmallerIsMoreCompact(t *testing.T) {
	content := "vmess://test.example.com:443"
	fullOut := generateQRCode(content, false)
	smallOut := generateQRCode(content, true)

	fullLines := strings.Split(strings.TrimRight(fullOut, "\n"), "\n")
	smallLines := strings.Split(strings.TrimRight(smallOut, "\n"), "\n")

	if len(smallLines) >= len(fullLines) {
		t.Errorf("smaller QR height %d should be <= full QR height %d", len(smallLines), len(fullLines))
	}
	if len([]rune(smallLines[0])) >= len([]rune(fullLines[0])) {
		t.Errorf("smaller QR width %d should be < full QR width %d",
			len([]rune(smallLines[0])), len([]rune(fullLines[0])))
	}
}

func verifyQuadrantChar(t *testing.T, got rune, ul, ur, ll, lr bool) {
	t.Helper()
	var idx int
	if ul {
		idx |= 8
	}
	if ur {
		idx |= 4
	}
	if ll {
		idx |= 2
	}
	if lr {
		idx |= 1
	}
	expected := quadrantChars[idx]
	if got != expected {
		t.Errorf("got %c (U+%04X), want %c (U+%04X) for ul=%v ur=%v ll=%v lr=%v",
			got, got, expected, expected, ul, ur, ll, lr)
	}
}
