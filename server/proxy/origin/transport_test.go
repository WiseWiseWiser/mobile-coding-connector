package origin

import (
	"context"
	"crypto/x509"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPSVerifiesCertificateAndNegotiatesHTTP2(t *testing.T) {
	up := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "tls-ok") }))
	up.EnableHTTP2 = true
	up.StartTLS()
	defer up.Close()
	transport := NewTransport()
	defer transport.Close()
	c := &http.Client{Transport: transport}
	if res, err := c.Get(up.URL); err == nil {
		res.Body.Close()
		t.Fatal("untrusted certificate accepted")
	}
	roots := x509.NewCertPool()
	roots.AddCert(up.Certificate())
	// Only the fixture CA is trusted; production never disables verification.
	trusted := NewTransport()
	defer trusted.Close()
	trusted.TLSClientConfig = up.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
	trusted.TLSClientConfig.InsecureSkipVerify = false
	trusted.TLSClientConfig.RootCAs = roots
	res, err := (&http.Client{Transport: trusted}).Get(up.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, res.Body)
	if res.ProtoMajor != 2 {
		t.Fatalf("HTTP/2 not negotiated: %s", res.Proto)
	}
}
func TestOriginDialRejectsMetadataAndSelfReference(t *testing.T) {
	tr := NewTransport()
	defer tr.Close()
	if _, err := tr.dial(context.Background(), "tcp", "169.254.169.254:80"); err == nil || !strings.Contains(err.Error(), "metadata") {
		t.Fatalf("metadata accepted: %v", err)
	}
	tr.Forbid("127.0.0.1:9876")
	if _, err := tr.dial(context.Background(), "tcp", "[::ffff:127.0.0.1]:9876"); err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("self-reference accepted: %v", err)
	}
}
