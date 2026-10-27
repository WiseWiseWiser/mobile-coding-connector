package screen

import "testing"

func TestDragButtonFor(t *testing.T) {
	cases := []struct {
		name string
		held map[byte]bool
		want byte
	}{
		{"nothing held is a hover", map[byte]bool{}, buttonNone},
		{"released button is a hover", map[byte]bool{buttonLeft: false}, buttonNone},
		{"left drags", map[byte]bool{buttonLeft: true}, buttonLeft},
		{"right drags", map[byte]bool{buttonRight: true}, buttonRight},
		{"middle drags", map[byte]bool{buttonMiddle: true}, buttonMiddle},
		{"left wins over right", map[byte]bool{buttonLeft: true, buttonRight: true}, buttonLeft},
		{"right wins over middle", map[byte]bool{buttonRight: true, buttonMiddle: true}, buttonRight},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := dragButtonFor(tc.held); got != tc.want {
				t.Fatalf("dragButtonFor(%v) = %d, want %d", tc.held, got, tc.want)
			}
		})
	}
}
