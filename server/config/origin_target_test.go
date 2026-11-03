package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOriginTarget(t *testing.T) {
	for _, tc := range []struct {
		host, scheme, want string
		bad                bool
	}{
		{"192.168.1.1", "", "http://192.168.1.1:3080", false},
		{"Macbook.local", "https", "https://macbook.local:3080", false},
		{"[2001:db8::1]", "", "http://[2001:db8::1]:3080", false},
		{"fe80::1234%eth0", "", "http://[fe80::1234%25eth0]:3080", false},
		{"fe80::1234", "", "", true}, {"http://host:9/path", "", "", true},
		{"user@host", "", "", true}, {"0.0.0.0", "", "", true},
		{"169.254.169.254", "", "", true}, {"::ffff:169.254.169.254", "", "", true},
		{"224.0.0.1", "", "", true}, {"host:9", "", "", true},
		{"host", "file", "", true},
	} {
		t.Run(tc.host+tc.scheme, func(t *testing.T) {
			got, err := (OriginTarget{Host: tc.host, Scheme: tc.scheme, Port: 3080}).Normalize(0)
			if (err != nil) != tc.bad {
				t.Fatalf("got %v", err)
			}
			if !tc.bad && got.URL() != tc.want {
				t.Fatalf("%s != %s", got.URL(), tc.want)
			}
		})
	}
}
func TestMappingStoreMigratesAndRetainsIDs(t *testing.T) {
	dir := t.TempDir()
	oldDesired, oldAuth, oldNames := PortMappingDesiredFile, PortMappingAuthFile, PortMappingNamesFile
	PortMappingDesiredFile = filepath.Join(dir, "port-mapping-desired.json")
	PortMappingAuthFile = filepath.Join(dir, "port-mapping-auth.json")
	PortMappingNamesFile = filepath.Join(dir, "port-mapping-names.json")
	defer func() {
		PortMappingDesiredFile, PortMappingAuthFile, PortMappingNamesFile = oldDesired, oldAuth, oldNames
	}()
	for path, text := range map[string]string{PortMappingDesiredFile: `{"3080":{"hostname":"pi.example","provider":"cloudflare_owned"}}`, PortMappingAuthFile: `{"3080":{"required":true,"token":"legacy-secret"}}`, PortMappingNamesFile: `{"3080":"pi.example","9090":"saved.example"}`} {
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	store, err := LoadMappings()
	if err != nil {
		t.Fatal(err)
	}
	old := store.Mappings[LocalMappingID(3080)]
	if !old.Desired || old.Target.Host != "127.0.0.1" || old.Auth.Token != "legacy-secret" {
		t.Fatalf("bad migration: %+v", old)
	}
	if store.Mappings[LocalMappingID(9090)].Desired {
		t.Fatal("saved name was published")
	}
	if _, err := os.Stat(PortMappingAuthFile + ".pre-target-v2"); err != nil {
		t.Fatal(err)
	}
	remote := MappingRecord{ID: "map_mac", Target: OriginTarget{Scheme: "http", Host: "192.168.1.1", Port: 3080}, Hostname: "mac.example", Desired: true}
	if err := UpdateMappings(func(s *MappingStore) error { s.Mappings[remote.ID] = remote; return nil }); err != nil {
		t.Fatal(err)
	}
	store, err = LoadMappings()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveMapping(store, "3080"); err == nil {
		t.Fatal("ambiguous port accepted")
	}
	if got, err := ResolveMapping(store, remote.ID); err != nil || got.Target.Host != "192.168.1.1" {
		t.Fatalf("ID lost: %+v %v", got, err)
	}
	info, err := os.Stat(MappingStorePath())
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("unsafe permissions")
	}
	if err := UpdateMappings(func(s *MappingStore) error {
		r := s.Mappings[old.ID]
		r.Desired = false
		s.Mappings[old.ID] = r
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	store, _ = LoadMappings()
	if store.Mappings[old.ID].Desired {
		t.Fatal("legacy desired resurrected disabled mapping")
	}
}
