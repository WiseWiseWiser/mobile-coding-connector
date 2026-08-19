package localiterm2

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReconcileSpaceLabelsKeepRematchDrop(t *testing.T) {
	stored := []*SpaceLabel{
		{SpaceID: 1, UUID: "aaa", Label: "Keep"},
		{SpaceID: 2, UUID: "bbb", Label: "Gone"},
		{SpaceID: 99, UUID: "ccc", Label: "Relink"},
	}
	live := []SpaceRef{
		{SpaceIndex: 0, SpaceID: 1, UUID: "aaa"},
		{SpaceIndex: 1, SpaceID: 7, UUID: "ccc"},
	}
	got, changed := reconcileSpaceLabels(stored, live)
	if !changed {
		t.Fatal("expected change")
	}
	if len(got) != 2 {
		t.Fatalf("len=%d want 2", len(got))
	}
	if got[0].Label != "Keep" || got[0].SpaceID != 1 {
		t.Fatalf("keep=%+v", got[0])
	}
	if got[1].Label != "Relink" || got[1].SpaceID != 7 {
		t.Fatalf("relink=%+v", got[1])
	}
}

func TestApplySpaceLabelsJoinsByID(t *testing.T) {
	inv := Inventory{
		Desktops: []DesktopGroup{
			{SpaceIndex: 0, Desktop: 1},
			{SpaceIndex: 1, Desktop: 2},
		},
	}
	live := []SpaceRef{
		{SpaceIndex: 0, SpaceID: 10, UUID: "u0"},
		{SpaceIndex: 1, SpaceID: 11, UUID: "u1", Current: true},
	}
	doc := &SpaceLabelsDocument{Labels: []*SpaceLabel{
		{SpaceID: 11, UUID: "u1", Label: "Review staging"},
	}}
	got := ApplySpaceLabels(inv, live, doc)
	if got.Desktops[0].SpaceID != 10 || got.Desktops[0].Label != "" {
		t.Fatalf("desk0=%+v", got.Desktops[0])
	}
	if got.Desktops[1].SpaceID != 11 || got.Desktops[1].Label != "Review staging" {
		t.Fatalf("desk1=%+v", got.Desktops[1])
	}
	if got.Desktops[0].Current {
		t.Fatal("desk0 should not be current")
	}
	if !got.Desktops[1].Current {
		t.Fatal("desk1 should be current")
	}
}

func TestApplySpaceLabelsDoesNotClip(t *testing.T) {
	inv := Inventory{
		Desktops: []DesktopGroup{
			{SpaceIndex: 0, Desktop: 1, Sessions: []LiveSession{{SessionID: "a"}}},
			{SpaceIndex: 15, Desktop: 16, Sessions: []LiveSession{{SessionID: "ghost16"}}},
		},
	}
	live := []SpaceRef{
		{SpaceIndex: 0, SpaceID: 10},
		{SpaceIndex: 1, SpaceID: 11},
	}
	got := ApplySpaceLabels(inv, live, emptySpaceLabels())
	if len(got.Desktops) != 2 {
		t.Fatalf("ApplySpaceLabels must not clip headings, got %d", len(got.Desktops))
	}
	if got.Desktops[1].SpaceIndex != 15 || got.Desktops[1].Sessions[0].SessionID != "ghost16" {
		t.Fatalf("ghost heading should remain until ClipDesktops: %+v", got.Desktops[1])
	}
}

func TestSpaceLabelStoreSetAndClear(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "space-labels.json")
	store := NewSpaceLabelStore(path)
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	if err := store.Set(10, "u0", "Review staging", now); err != nil {
		t.Fatal(err)
	}
	doc := store.Document()
	if len(doc.Labels) != 1 || doc.Labels[0].Label != "Review staging" {
		t.Fatalf("doc=%+v", doc)
	}
	if err := store.Set(10, "u0", "  ", now); err != nil {
		t.Fatal(err)
	}
	doc = store.Document()
	if len(doc.Labels) != 0 {
		t.Fatalf("cleared still has %d", len(doc.Labels))
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestHandleSpaceLabelsPutAndMissingID(t *testing.T) {
	dir := t.TempDir()
	h := &Handler{
		SpaceLabels: NewSpaceLabelStore(filepath.Join(dir, "space-labels.json")),
		ListSpaces: func() ([]SpaceRef, error) {
			return []SpaceRef{{SpaceIndex: 0, SpaceID: 10, UUID: "u0"}}, nil
		},
		CountDesktops: func() (int, error) { return 1, nil },
	}

	body, _ := json.Marshal(map[string]any{"space_id": 10, "uuid": "u0", "label": "Review staging"})
	req := httptest.NewRequest(http.MethodPut, SpaceLabelsPath, bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.handleSpaceLabels(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("set status=%d body=%s", rec.Code, rec.Body.String())
	}

	bad := httptest.NewRequest(http.MethodPut, SpaceLabelsPath, bytes.NewReader([]byte(`{"label":"x"}`)))
	badRec := httptest.NewRecorder()
	h.handleSpaceLabels(badRec, bad)
	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("missing id status=%d", badRec.Code)
	}

	clearBody, _ := json.Marshal(map[string]any{"space_id": 10, "label": ""})
	clearReq := httptest.NewRequest(http.MethodPut, SpaceLabelsPath, bytes.NewReader(clearBody))
	clearRec := httptest.NewRecorder()
	h.handleSpaceLabels(clearRec, clearReq)
	if clearRec.Code != http.StatusOK {
		t.Fatalf("clear status=%d", clearRec.Code)
	}
	if n := len(h.SpaceLabels.Document().Labels); n != 0 {
		t.Fatalf("after clear n=%d", n)
	}
}
