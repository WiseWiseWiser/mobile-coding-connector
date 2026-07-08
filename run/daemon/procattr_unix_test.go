//go:build unix

package daemon

import "testing"

func TestServerChildProcAttrDetach(t *testing.T) {
	attached := serverChildProcAttr(false)
	if attached.Setsid {
		t.Fatal("attached child should not use Setsid")
	}
	if !attached.Setpgid {
		t.Fatal("attached child should use Setpgid")
	}

	detached := serverChildProcAttr(true)
	if !detached.Setsid {
		t.Fatal("detached child should use Setsid")
	}
	if !detached.Setpgid {
		t.Fatal("detached child should use Setpgid")
	}
}