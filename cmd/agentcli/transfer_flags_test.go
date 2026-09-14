package agentcli

import "testing"

func TestParseUploadFlags(t *testing.T) {
	dry, noComp, noOverride, rest := parseUploadFlags([]string{"--dry-run", "--no-compress", "--no-override", "./a", "/tmp/a"})
	if !dry || !noComp || !noOverride {
		t.Fatalf("dry=%v noCompress=%v noOverride=%v", dry, noComp, noOverride)
	}
	if len(rest) != 2 || rest[0] != "./a" || rest[1] != "/tmp/a" {
		t.Fatalf("rest=%v", rest)
	}
	dry, noComp, noOverride, rest = parseUploadFlags([]string{"./b"})
	if dry || noComp || noOverride || len(rest) != 1 || rest[0] != "./b" {
		t.Fatalf("got dry=%v noCompress=%v noOverride=%v rest=%v", dry, noComp, noOverride, rest)
	}
}
