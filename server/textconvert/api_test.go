package textconvert

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func serve(h *Handler, req *http.Request) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	Register(mux, h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestConvertEndpointOK(t *testing.T) {
	body := `{"text":"a \\\n  --b"}`
	rec := serve(&Handler{}, httptest.NewRequest(http.MethodPost, ConvertPath, strings.NewReader(body)))
	if rec.Code != 200 {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var got ConvertResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Text != "a --b" {
		t.Fatalf("text %q", got.Text)
	}
	if got.Converter != ConverterShellSingleLine {
		t.Fatalf("converter %q", got.Converter)
	}
}

func TestConvertEndpointMalformedJSON(t *testing.T) {
	rec := serve(&Handler{}, httptest.NewRequest(http.MethodPost, ConvertPath, strings.NewReader(`{`)))
	if rec.Code != 400 {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestConvertEndpointUnknownConverter(t *testing.T) {
	body := `{"text":"x","converter":"nope"}`
	rec := serve(&Handler{}, httptest.NewRequest(http.MethodPost, ConvertPath, strings.NewReader(body)))
	if rec.Code != 400 {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var errBody map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
		t.Fatal(err)
	}
	msg, _ := errBody["error"].(string)
	if !strings.Contains(msg, "unknown converter") {
		t.Fatalf("error %q", msg)
	}
}

func TestConvertEndpointMethodNotAllowed(t *testing.T) {
	rec := serve(&Handler{}, httptest.NewRequest(http.MethodGet, ConvertPath, nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status %d", rec.Code)
	}
}
