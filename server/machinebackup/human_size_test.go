package machinebackup

import "testing"

func TestParseHumanSize(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"40MB", 40 * 1024 * 1024},
		{"50M", 50 * 1024 * 1024},
		{"1G", 1024 * 1024 * 1024},
		{"1GB", 1024 * 1024 * 1024},
		{"100MB", 100 * 1024 * 1024},
		{"512K", 512 * 1024},
		{"0B", 0},
	}
	for _, tc := range cases {
		got, err := ParseHumanSize(tc.in)
		if err != nil {
			t.Fatalf("ParseHumanSize(%q) err = %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("ParseHumanSize(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestParseHumanSizeInvalid(t *testing.T) {
	for _, in := range []string{"", "nope", "-5MB"} {
		if _, err := ParseHumanSize(in); err == nil {
			t.Fatalf("ParseHumanSize(%q) want error", in)
		}
	}
}

func TestEffectiveLargeDirThreshold(t *testing.T) {
	if got := EffectiveLargeDirThreshold(0); got != defaultLargeDirThresholdBytes {
		t.Fatalf("default = %d, want %d", got, defaultLargeDirThresholdBytes)
	}
	custom := int64(100 * 1024 * 1024)
	if got := EffectiveLargeDirThreshold(custom); got != custom {
		t.Fatalf("custom = %d, want %d", got, custom)
	}
}

func TestResolveLargeDirThresholdBytes(t *testing.T) {
	home := t.TempDir()
	got, err := ResolveLargeDirThresholdBytes(home, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got != defaultLargeDirThresholdBytes {
		t.Fatalf("default = %d, want %d", got, defaultLargeDirThresholdBytes)
	}

	cli := int64(50 * 1024 * 1024)
	got, err = ResolveLargeDirThresholdBytes(home, cli)
	if err != nil || got != cli {
		t.Fatalf("CLI override = %d, err = %v, want %d", got, err, cli)
	}

	if err := SaveUserBackupConfig(home, nil, "100MB"); err != nil {
		t.Fatal(err)
	}
	got, err = ResolveLargeDirThresholdBytes(home, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := int64(100 * 1024 * 1024)
	if got != want {
		t.Fatalf("persisted = %d, want %d", got, want)
	}
}