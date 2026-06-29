package projects

import "testing"

func TestGitStatusDirty(t *testing.T) {
	cases := []struct {
		status GitStatusInfo
		want   bool
	}{
		{GitStatusInfo{Commit: "abc1234", IsClean: true}, false},
		{GitStatusInfo{Commit: "abc1234", IsClean: false}, true},
		{GitStatusInfo{IsClean: true}, false},
		{GitStatusInfo{IsClean: false}, false},
	}
	for _, tc := range cases {
		if got := gitStatusDirty(tc.status); got != tc.want {
			t.Fatalf("gitStatusDirty(%+v) = %v, want %v", tc.status, got, tc.want)
		}
	}
}