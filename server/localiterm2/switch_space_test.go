package localiterm2

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleSwitchSpaceOKAndMissingID(t *testing.T) {
	var got uint64
	h := &Handler{
		SwitchSpace: func(spaceID uint64) error {
			got = spaceID
			return nil
		},
	}

	body, _ := json.Marshal(map[string]any{"space_id": 2291})
	req := httptest.NewRequest(http.MethodPost, SwitchSpacePath, bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.handleSwitchSpace(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got != 2291 {
		t.Fatalf("got space_id=%d", got)
	}

	bad := httptest.NewRequest(http.MethodPost, SwitchSpacePath, bytes.NewReader([]byte(`{}`)))
	badRec := httptest.NewRecorder()
	h.handleSwitchSpace(badRec, bad)
	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("missing id status=%d", badRec.Code)
	}

	fail := &Handler{
		SwitchSpace: func(spaceID uint64) error {
			return fmt.Errorf("active space still 1 after switch to %d", spaceID)
		},
	}
	failReq := httptest.NewRequest(http.MethodPost, SwitchSpacePath, bytes.NewReader(body))
	failRec := httptest.NewRecorder()
	fail.handleSwitchSpace(failRec, failReq)
	if failRec.Code != http.StatusInternalServerError {
		t.Fatalf("fail status=%d", failRec.Code)
	}

	mux := http.NewServeMux()
	Register(mux, h)
	mountReq := httptest.NewRequest(http.MethodPost, SwitchSpacePath, bytes.NewReader(body))
	mountRec := httptest.NewRecorder()
	mux.ServeHTTP(mountRec, mountReq)
	if mountRec.Code == http.StatusNotFound {
		t.Fatal("switch-space not mounted")
	}
}
