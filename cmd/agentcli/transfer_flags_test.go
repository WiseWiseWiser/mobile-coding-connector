package agentcli

import "testing"

func TestParseUploadFlags(t *testing.T) {
	dry, noComp, rest := parseUploadFlags([]string{"--dry-run", "--no-compress", "./a", "/tmp/a"})
	if !dry || !noComp {
		t.Fatalf("dry=%v noCompress=%v", dry, noComp)
	}
	if len(rest) != 2 || rest[0] != "./a" || rest[1] != "/tmp/a" {
		t.Fatalf("rest=%v", rest)
	}
	dry, noComp, rest = parseUploadFlags([]string{"./b"})
	if dry || noComp || len(rest) != 1 || rest[0] != "./b" {
		t.Fatalf("got dry=%v noCompress=%v rest=%v", dry, noComp, rest)
	}
}
