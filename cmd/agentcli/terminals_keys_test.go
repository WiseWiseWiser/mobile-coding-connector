package agentcli

import (
	"bufio"
	"strings"
	"testing"
)

func TestReadTerminalsKey(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want []string
	}{
		{"q", []string{"q"}},
		{"\r", []string{"enter"}},
		{"\t", []string{"tab"}},
		{"jkl", []string{"j", "k", "l"}},
		{"\x1b[A\x1b[B\x1b[C\x1b[D", []string{"up", "down", "right", "left"}},
		{"\x1b", []string{"esc"}},
		{"\x03", []string{"q"}},
	}
	for _, tc := range cases {
		br := bufio.NewReader(strings.NewReader(tc.in))
		var got []string
		for {
			k, err := readTerminalsKey(br, nil)
			if err != nil {
				break
			}
			if k != "" {
				got = append(got, k)
			}
		}
		if len(got) != len(tc.want) {
			t.Fatalf("in %q got %v want %v", tc.in, got, tc.want)
		}
		for i := range tc.want {
			if got[i] != tc.want[i] {
				t.Fatalf("in %q [%d] got %q want %q", tc.in, i, got[i], tc.want[i])
			}
		}
	}
}
