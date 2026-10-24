package config

import (
	"path/filepath"
	"testing"
)

func useTempPortMappingDesiredFile(t *testing.T) {
	t.Helper()
	oldFile := PortMappingDesiredFile
	oldDir := DataDir
	dir := t.TempDir()
	DataDir = dir
	PortMappingDesiredFile = filepath.Join(dir, "port-mapping-desired.json")
	t.Cleanup(func() {
		PortMappingDesiredFile = oldFile
		DataDir = oldDir
	})
}

func TestPortMappingDesiredRoundTrip(t *testing.T) {
	useTempPortMappingDesiredFile(t)
	got, err := LoadPortMappingDesired()
	if err != nil || len(got) != 0 {
		t.Fatalf("empty load = %#v, %v", got, err)
	}
	if err := SetPortMappingDesired(8422, PortMappingDesiredSpec{
		Hostname: " eluc-on-mac.xhd2015.xyz ",
		Provider: " cloudflare_owned ",
	}); err != nil {
		t.Fatal(err)
	}
	got, err = LoadPortMappingDesired()
	if err != nil {
		t.Fatal(err)
	}
	spec := got["8422"]
	if spec.Hostname != "eluc-on-mac.xhd2015.xyz" || spec.Provider != "cloudflare_owned" {
		t.Fatalf("saved = %#v", spec)
	}
	if err := DeletePortMappingDesired(8422); err != nil {
		t.Fatal(err)
	}
	got, err = LoadPortMappingDesired()
	if err != nil || len(got) != 0 {
		t.Fatalf("after delete = %#v, %v", got, err)
	}
}

func TestDeletePortMappingDesiredMissingIsOK(t *testing.T) {
	useTempPortMappingDesiredFile(t)
	if err := DeletePortMappingDesired(9); err != nil {
		t.Fatal(err)
	}
}

func TestSetPortMappingDesiredRequiresHostname(t *testing.T) {
	useTempPortMappingDesiredFile(t)
	if err := SetPortMappingDesired(8422, PortMappingDesiredSpec{}); err == nil {
		t.Fatal("expected hostname required")
	}
}
