package agentcli

import (
	"strings"
	"testing"
)

func TestParseVpnFlagsAlsoProxyRequiresHTTPOnly(t *testing.T) {
	_, err := parseVpnFlags([]string{"--also-proxy", "git.example.com:22"}, vpnHelp)
	if err == nil {
		t.Fatal("expected error without --http-only")
	}
	if !strings.Contains(err.Error(), "--also-proxy requires --http-only") {
		t.Fatalf("err=%v", err)
	}
}

func TestParseVpnFlagsAlsoProxyOK(t *testing.T) {
	opts, err := parseVpnFlags([]string{
		"--http-only",
		"--dns-hijack",
		"--also-proxy", "git.example.com:22",
		"--also-proxy", "*.db.internal:6606",
	}, vpnHelp)
	if err != nil {
		t.Fatalf("parseVpnFlags: %v", err)
	}
	if !opts.HttpOnly || !opts.DNSHijack {
		t.Fatalf("flags: HttpOnly=%v DNSHijack=%v", opts.HttpOnly, opts.DNSHijack)
	}
	if len(opts.AlsoProxy) != 2 {
		t.Fatalf("AlsoProxy len=%d want 2: %+v", len(opts.AlsoProxy), opts.AlsoProxy)
	}
	if opts.AlsoProxy[0].Port != 22 || opts.AlsoProxy[0].Host != "git.example.com" {
		t.Fatalf("AlsoProxy[0]=%+v", opts.AlsoProxy[0])
	}
	if opts.AlsoProxy[1].Port != 6606 || opts.AlsoProxy[1].Host != ".db.internal" {
		t.Fatalf("AlsoProxy[1]=%+v", opts.AlsoProxy[1])
	}
}

func TestParseVpnFlagsAlsoProxyPortOnly(t *testing.T) {
	_, err := parseVpnFlags([]string{"--http-only", "--also-proxy", ":22"}, vpnHelp)
	if err == nil {
		t.Fatal("expected error for :22")
	}
	if !strings.Contains(err.Error(), "port-only") {
		t.Fatalf("err=%v", err)
	}
}

func TestParseVpnFlagsRemoteDirect(t *testing.T) {
	opts, err := parseVpnFlags([]string{
		"--http-only",
		"--dns-hijack",
		"--also-proxy", "*.db.internal:6606",
		"--remote-direct", "*.db.internal:6606",
	}, vpnHelp)
	if err != nil {
		t.Fatalf("parseVpnFlags: %v", err)
	}
	if len(opts.RemoteDirect) != 1 {
		t.Fatalf("RemoteDirect=%+v", opts.RemoteDirect)
	}
	if opts.RemoteDirect[0].Port != 6606 || opts.RemoteDirect[0].Host != ".db.internal" {
		t.Fatalf("RemoteDirect[0]=%+v", opts.RemoteDirect[0])
	}
}

func TestParseVpnFlagsRemoteDirectPortOnly(t *testing.T) {
	_, err := parseVpnFlags([]string{"--remote-direct", ":6606"}, vpnHelp)
	if err == nil || !strings.Contains(err.Error(), "port-only") {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(err.Error(), "--remote-direct") {
		t.Fatalf("error should mention --remote-direct: %v", err)
	}
}
