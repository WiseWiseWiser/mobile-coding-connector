package usageitems

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestService(t *testing.T) (*Service, *Store) {
	t.Helper()
	store := NewStore(filepath.Join(t.TempDir(), "usage-items.json"))
	return NewService(store), store
}

// readyFetcher answers every item with the same ready snapshot.
func readyFetcher(svc *Service) {
	TestExported_SetSnapshotFetcher(svc, func(it Item) (Snapshot, error) {
		return Snapshot{
			Status:    StatusReady,
			Percent:   "5%",
			Body:      "5% used, $66.19 left",
			UsageURL:  "https://commandcode.ai/acct/settings/usage",
			UpdatedAt: "2026-09-15T12:00:00Z",
		}, nil
	})
}

func errorKind(t *testing.T, err error) ErrorKind {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error")
	}
	var itemErr *ItemError
	if !errors.As(err, &itemErr) {
		t.Fatalf("error %v is not an ItemError", err)
	}
	return itemErr.Kind
}

func TestStoreSeedsGrokAndCodex(t *testing.T) {
	svc, store := newTestService(t)

	reg, err := svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	if got := reg.SortedIDs(); len(got) != 2 || got[0] != KindCodex || got[1] != KindGrok {
		t.Fatalf("seeded ids = %v, want [codex grok]", got)
	}
	if reg.Default != KindGrok {
		t.Fatalf("default = %q, want grok", reg.Default)
	}
	if !reg.Rotate {
		t.Fatal("seeded registry must rotate")
	}
	if reg.Version != RegistryVersion {
		t.Fatalf("version = %d, want %d", reg.Version, RegistryVersion)
	}
	if _, err := os.Stat(store.Path()); err != nil {
		t.Fatalf("seed must be written to %s: %v", store.Path(), err)
	}

	// A second read reuses the file rather than re-seeding.
	reg.Items = append(reg.Items, Item{ID: "extra", Label: "Extra", Kind: KindGrok})
	if err := store.Save(reg); err != nil {
		t.Fatalf("save: %v", err)
	}
	again, err := svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	if len(again.Items) != 3 {
		t.Fatalf("reloaded items = %d, want 3", len(again.Items))
	}
}

func TestListRendersSeededItems(t *testing.T) {
	svc, _ := newTestService(t)
	readyFetcher(svc)

	list, err := svc.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(list.Items))
	}
	views := map[string]ItemView{}
	for _, v := range list.Items {
		views[v.ID] = v
	}
	grok := views[KindGrok]
	if grok.Status != StatusReady {
		t.Fatalf("grok status = %q", grok.Status)
	}
	if grok.Title != "Grok 5%" {
		t.Fatalf("grok title = %q, want %q", grok.Title, "Grok 5%")
	}
	if grok.Dropdown != "Grok: 5% used, $66.19 left" {
		t.Fatalf("grok dropdown = %q", grok.Dropdown)
	}
	if list.Default != KindGrok || !list.Rotate {
		t.Fatalf("list selection = %q/%v", list.Default, list.Rotate)
	}
}

func TestShowUnknownItem(t *testing.T) {
	svc, _ := newTestService(t)
	_, err := svc.Show("nope")
	if got := errorKind(t, err); got != ErrorNotFound {
		t.Fatalf("kind = %q, want not_found", got)
	}
	if !strings.Contains(err.Error(), "known items: grok, codex") {
		t.Fatalf("error should list known ids: %v", err)
	}
}

func TestAddDerivesIDAndSuffixesTakenLabel(t *testing.T) {
	svc, _ := newTestService(t)
	readyFetcher(svc)

	first, warnings, err := svc.Add(AddRequest{Item: Item{Label: "CC v1", Kind: KindCommandCode, Home: "/tmp/cc1"}})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if first.ID != "cc-v1" {
		t.Fatalf("id = %q, want cc-v1", first.ID)
	}
	if !first.Enabled {
		t.Fatal("added item must default to enabled")
	}

	second, _, err := svc.Add(AddRequest{Item: Item{ID: "cc-v2", Kind: KindCommandCode, Home: "/tmp/cc2"}})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if second.Label != "CommandCode" {
		t.Fatalf("first default label = %q, want CommandCode", second.Label)
	}

	third, _, err := svc.Add(AddRequest{Item: Item{ID: "cc-v3", Kind: KindCommandCode, Home: "/tmp/cc3"}})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if third.Label != "CommandCode cc-v3" {
		t.Fatalf("colliding default label = %q, want %q", third.Label, "CommandCode cc-v3")
	}
	if third.Title != "CommandCode cc-v3 5%" {
		t.Fatalf("title = %q", third.Title)
	}

	reg, err := svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	if got := reg.SortedIDs(); strings.Join(got, ",") != "cc-v1,cc-v2,cc-v3,codex,grok" {
		t.Fatalf("ids = %v", got)
	}
}

func TestAddRequiresKindFields(t *testing.T) {
	svc, _ := newTestService(t)
	readyFetcher(svc)

	_, _, err := svc.Add(AddRequest{Item: Item{ID: "cc", Kind: KindCommandCode}})
	if got := errorKind(t, err); got != ErrorInvalid {
		t.Fatalf("kind = %q, want invalid", got)
	}
	if !strings.Contains(err.Error(), "--home is required for kind commandcode") {
		t.Fatalf("error = %v", err)
	}

	_, _, err = svc.Add(AddRequest{Item: Item{ID: "x", Kind: "nope"}})
	if got := errorKind(t, err); got != ErrorInvalid {
		t.Fatalf("kind = %q, want invalid", got)
	}

	_, _, err = svc.Add(AddRequest{Item: Item{Kind: KindGrok}})
	if got := errorKind(t, err); got != ErrorInvalid {
		t.Fatalf("kind = %q, want invalid", got)
	}
	if !strings.Contains(err.Error(), "--id or --label is required") {
		t.Fatalf("error = %v", err)
	}

	_, _, err = svc.Add(AddRequest{Item: Item{ID: KindGrok, Kind: KindGrok}})
	if got := errorKind(t, err); got != ErrorConflict {
		t.Fatalf("kind = %q, want conflict", got)
	}
}

func TestAddValidatesProviderUnlessStrict(t *testing.T) {
	svc, _ := newTestService(t)
	TestExported_SetSnapshotFetcher(svc, func(it Item) (Snapshot, error) {
		if strings.HasPrefix(it.ID, "cc-bad") {
			return Snapshot{}, fmt.Errorf("401 unauthorized")
		}
		return Snapshot{Status: StatusReady, Percent: "5%", Body: "5% used"}, nil
	})

	view, warnings, err := svc.Add(AddRequest{Item: Item{ID: "cc-bad", Kind: KindCommandCode, Home: "/tmp/cc"}})
	if err != nil {
		t.Fatalf("add must warn, not fail: %v", err)
	}
	if len(warnings) != 1 || warnings[0] != "cc-bad fetch failed: 401 unauthorized" {
		t.Fatalf("warnings = %v", warnings)
	}
	if view.Status != StatusError || view.Error != "401 unauthorized" {
		t.Fatalf("view = %+v", view)
	}
	if view.Title != "CommandCode err" {
		t.Fatalf("error title = %q", view.Title)
	}
	if view.Dropdown != "CommandCode: Error: 401 unauthorized" {
		t.Fatalf("error dropdown = %q", view.Dropdown)
	}

	reg, err := svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	if reg.Find("cc-bad") == nil {
		t.Fatal("warned item must still be registered")
	}

	_, _, err = svc.Add(AddRequest{Item: Item{ID: "cc-bad2", Kind: KindCommandCode, Home: "/tmp/cc"}, Strict: true})
	if got := errorKind(t, err); got != ErrorInvalid {
		t.Fatalf("kind = %q, want invalid", got)
	}
	if !strings.Contains(err.Error(), "cc-bad2 fetch failed: 401 unauthorized") {
		t.Fatalf("error = %v", err)
	}
	reg, err = svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	if reg.Find("cc-bad2") != nil {
		t.Fatal("strict failure must not register the item")
	}
}

func TestAddSkipValidateIgnoresProvider(t *testing.T) {
	svc, _ := newTestService(t)
	calls := 0
	TestExported_SetSnapshotFetcher(svc, func(it Item) (Snapshot, error) {
		calls++
		return Snapshot{}, fmt.Errorf("boom")
	})

	_, warnings, err := svc.Add(AddRequest{
		Item:         Item{ID: "cc", Kind: KindCommandCode, Home: "/tmp/cc"},
		SkipValidate: true,
		Strict:       true,
	})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if calls != 0 {
		t.Fatalf("fetch calls = %d, want 0", calls)
	}
}

func TestAddAsDefaultClaimsMenuBar(t *testing.T) {
	svc, _ := newTestService(t)
	readyFetcher(svc)

	if _, _, err := svc.Add(AddRequest{
		Item:    Item{ID: "cc-v1", Kind: KindCommandCode, Home: "/tmp/cc1"},
		Default: true,
	}); err != nil {
		t.Fatalf("add: %v", err)
	}
	reg, err := svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	if reg.Default != "cc-v1" {
		t.Fatalf("default = %q, want cc-v1", reg.Default)
	}
	if reg.Rotate {
		t.Fatal("choosing a default must stop rotation")
	}
}

func TestUpdateKeepsLabelAndRejectsBadKind(t *testing.T) {
	svc, _ := newTestService(t)
	readyFetcher(svc)

	if _, _, err := svc.Add(AddRequest{Item: Item{ID: "cc-v1", Kind: KindCommandCode, Home: "/tmp/cc1"}}); err != nil {
		t.Fatalf("add: %v", err)
	}

	label := "CC Sandbox v1"
	view, warnings, err := svc.Update(UpdateRequest{ID: "cc-v1", Update: ItemUpdate{Label: &label}})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if view.Label != label || view.Kind != KindCommandCode {
		t.Fatalf("view = %+v", view)
	}
	if view.Home != "/tmp/cc1" {
		t.Fatalf("home = %q, want unchanged", view.Home)
	}
	if view.Title != "CC Sandbox v1 5%" {
		t.Fatalf("title = %q", view.Title)
	}

	kind := KindCommandCode
	if _, _, err := svc.Update(UpdateRequest{ID: "cc-v1", Update: ItemUpdate{Kind: &kind}}); err != nil {
		t.Fatalf("same-kind update: %v", err)
	}

	blank := ""
	if _, _, err := svc.Update(UpdateRequest{ID: "cc-v1", Update: ItemUpdate{Kind: &blank}}); errorKind(t, err) != ErrorInvalid {
		t.Fatalf("blank kind must be rejected: %v", err)
	}

	unknown := "nope"
	_, _, err = svc.Update(UpdateRequest{ID: "cc-v1", Update: ItemUpdate{Kind: &unknown}})
	if got := errorKind(t, err); got != ErrorInvalid {
		t.Fatalf("kind = %q, want invalid", got)
	}

	_, _, err = svc.Update(UpdateRequest{ID: "missing", Update: ItemUpdate{Label: &label}})
	if got := errorKind(t, err); got != ErrorNotFound {
		t.Fatalf("kind = %q, want not_found", got)
	}
}

func TestUpdateValidatesWhenProviderBindingChanges(t *testing.T) {
	svc, _ := newTestService(t)
	fetchCalls := 0
	TestExported_SetSnapshotFetcher(svc, func(it Item) (Snapshot, error) {
		fetchCalls++
		if it.Home == "/tmp/dead" {
			return Snapshot{}, fmt.Errorf("connection refused")
		}
		return Snapshot{Status: StatusReady, Percent: "5%", Body: "5% used"}, nil
	})

	if _, _, err := svc.Add(AddRequest{Item: Item{ID: "cc-v1", Kind: KindCommandCode, Home: "/tmp/cc1"}, SkipValidate: true}); err != nil {
		t.Fatalf("add: %v", err)
	}
	before := fetchCalls

	dead := "/tmp/dead"
	_, warnings, err := svc.Update(UpdateRequest{ID: "cc-v1", Update: ItemUpdate{Home: &dead}})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(warnings) != 1 || warnings[0] != "cc-v1 fetch failed: connection refused" {
		t.Fatalf("warnings = %v", warnings)
	}
	if fetchCalls == before {
		t.Fatal("changing --home must re-validate")
	}

	// A warning is not a rejection: the new home is persisted.
	reg, err := svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	if it := reg.Find("cc-v1"); it == nil || it.Home != dead {
		t.Fatalf("item = %+v, want home %q", it, dead)
	}

	// A label-only change keeps the provider binding, so it warns about nothing.
	label := "Renamed"
	_, warnings, err = svc.Update(UpdateRequest{ID: "cc-v1", Update: ItemUpdate{Label: &label}})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("label-only warnings = %v", warnings)
	}
}

func TestUpdateStrictFailsOnProviderError(t *testing.T) {
	svc, _ := newTestService(t)
	TestExported_SetSnapshotFetcher(svc, func(it Item) (Snapshot, error) {
		if it.Home == "/tmp/dead" {
			return Snapshot{}, fmt.Errorf("connection refused")
		}
		return Snapshot{Status: StatusReady, Percent: "5%", Body: "5% used"}, nil
	})

	if _, _, err := svc.Add(AddRequest{Item: Item{ID: "cc-v1", Kind: KindCommandCode, Home: "/tmp/cc1"}, SkipValidate: true}); err != nil {
		t.Fatalf("add: %v", err)
	}

	dead := "/tmp/dead"
	_, _, err := svc.Update(UpdateRequest{ID: "cc-v1", Update: ItemUpdate{Home: &dead}, Strict: true})
	if got := errorKind(t, err); got != ErrorInvalid {
		t.Fatalf("kind = %q, want invalid", got)
	}
	reg, err := svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	if it := reg.Find("cc-v1"); it == nil || it.Home != "/tmp/cc1" {
		t.Fatalf("strict failure must leave the item untouched: %+v", it)
	}
}

func TestSetDefaultStopsRotationAndResumes(t *testing.T) {
	svc, _ := newTestService(t)
	readyFetcher(svc)

	list, err := svc.SetDefault(KindCodex, false)
	if err != nil {
		t.Fatalf("set default: %v", err)
	}
	if list.Default != KindCodex || list.Rotate {
		t.Fatalf("selection = %q/%v, want codex/false", list.Default, list.Rotate)
	}

	list, err = svc.SetDefault("", true)
	if err != nil {
		t.Fatalf("set default: %v", err)
	}
	if !list.Rotate {
		t.Fatal("rotate must be re-enabled")
	}
	if list.Default != KindCodex {
		t.Fatalf("default while rotating = %q, want codex kept", list.Default)
	}

	_, err = svc.SetDefault("missing", false)
	if got := errorKind(t, err); got != ErrorNotFound {
		t.Fatalf("kind = %q, want not_found", got)
	}
}

func TestRotateCoversEnabledItemsInRegistryOrder(t *testing.T) {
	svc, _ := newTestService(t)
	readyFetcher(svc)

	if _, _, err := svc.Add(AddRequest{Item: Item{ID: "cc-v1", Kind: KindCommandCode, Home: "/tmp/cc1"}}); err != nil {
		t.Fatalf("add: %v", err)
	}

	reg, err := svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	var ids []string
	for _, it := range reg.EnabledItems() {
		ids = append(ids, it.ID)
	}
	if strings.Join(ids, ",") != "grok,codex,cc-v1" {
		t.Fatalf("rotation order = %v, want [grok codex cc-v1]", ids)
	}

	no := false
	if _, _, err := svc.Update(UpdateRequest{ID: KindCodex, Update: ItemUpdate{Enabled: &no}, SkipValidate: true}); err != nil {
		t.Fatalf("update: %v", err)
	}
	reg, err = svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	ids = nil
	for _, it := range reg.EnabledItems() {
		ids = append(ids, it.ID)
	}
	if strings.Join(ids, ",") != "grok,cc-v1" {
		t.Fatalf("rotation order after disabling codex = %v", ids)
	}
}

func TestRemoveReassignsDefault(t *testing.T) {
	svc, _ := newTestService(t)
	readyFetcher(svc)

	if _, _, err := svc.Add(AddRequest{Item: Item{ID: "cc-v1", Kind: KindCommandCode, Home: "/tmp/cc1"}}); err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, err := svc.SetDefault("cc-v1", false); err != nil {
		t.Fatalf("set default: %v", err)
	}

	res, err := svc.Remove("cc-v1")
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if res.Removed != "cc-v1" || res.Default != KindGrok {
		t.Fatalf("result = %+v, want removed cc-v1 with default grok", res)
	}
	if len(res.Warnings) != 1 || res.Warnings[0] != "cc-v1 was the menu bar default; default is now grok" {
		t.Fatalf("warnings = %v", res.Warnings)
	}

	// A non-default removal is silent.
	res, err = svc.Remove(KindCodex)
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if len(res.Warnings) != 0 {
		t.Fatalf("warnings = %v", res.Warnings)
	}
	if res.Default != KindGrok {
		t.Fatalf("default = %q, want grok", res.Default)
	}

	// Removing the last item clears the default.
	if _, err := svc.Remove(KindGrok); err != nil {
		t.Fatalf("remove: %v", err)
	}
	reg, err := svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	if len(reg.Items) != 0 || reg.Default != "" {
		t.Fatalf("registry = %+v, want empty", reg)
	}

	if _, err := svc.Remove("grok"); errorKind(t, err) != ErrorNotFound {
		t.Fatalf("removing an unknown item: %v", err)
	}
}

func TestShowFetchesAndRendersDetail(t *testing.T) {
	svc, _ := newTestService(t)
	var fetched []string
	TestExported_SetSnapshotFetcher(svc, func(it Item) (Snapshot, error) {
		fetched = append(fetched, it.ID)
		if it.ID == KindGrok {
			return Snapshot{}, fmt.Errorf("grok offline")
		}
		return Snapshot{
			Status:   StatusReady,
			Percent:  "5%",
			Body:     "5% used, $66.19 left, 1,870 requests, 5h 15%, Weekly 11%, renews in 26d",
			Detail:   "GOAT Plan · active\n5% used",
			UsageURL: "https://commandcode.ai/acct/settings/usage",
		}, nil
	})

	if _, warnings, err := svc.Add(AddRequest{Item: Item{ID: "cc-v1", Kind: KindCommandCode, Home: "/tmp/cc1"}}); err != nil {
		t.Fatalf("add: %v", err)
	} else if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}

	view, err := svc.Show("cc-v1")
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	if view.Title != "CommandCode 5%" {
		t.Fatalf("title = %q", view.Title)
	}
	wantBody := "5% used, $66.19 left, 1,870 requests, 5h 15%, Weekly 11%, renews in 26d"
	if view.Dropdown != "CommandCode: "+wantBody {
		t.Fatalf("dropdown = %q", view.Dropdown)
	}
	if view.Detail == "" || view.UsageURL == "" {
		t.Fatalf("view must carry the panel: %+v", view)
	}

	show, err := svc.Show(KindGrok)
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	if show.Status != StatusError {
		t.Fatalf("grok status = %q", show.Status)
	}
	if show.Title != "Grok err" || show.Dropdown != "Grok: Error: grok offline" {
		t.Fatalf("error view = %+v", show)
	}
}

func TestLoadingItemsRenderPlaceholders(t *testing.T) {
	svc, _ := newTestService(t)
	TestExported_SetSnapshotFetcher(svc, func(it Item) (Snapshot, error) {
		return Snapshot{Status: StatusLoading}, nil
	})

	view, err := svc.Show(KindGrok)
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	if view.Title != "Grok ..." {
		t.Fatalf("title = %q, want %q", view.Title, "Grok ...")
	}
	if view.Dropdown != "Grok: Loading..." {
		t.Fatalf("dropdown = %q", view.Dropdown)
	}
}

func TestTitleTruncatesLongLabels(t *testing.T) {
	svc, _ := newTestService(t)
	readyFetcher(svc)

	long := strings.Repeat("Command Code ", 6)
	if _, _, err := svc.Add(AddRequest{
		Item:         Item{ID: "cc-long", Kind: KindCommandCode, Home: "/tmp/cc", Label: long},
		SkipValidate: true,
	}); err != nil {
		t.Fatalf("add: %v", err)
	}

	view, err := svc.Show("cc-long")
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	if n := len([]rune(view.Title)); n > MaxLabelLen {
		t.Fatalf("title has %d runes, cap is %d: %q", n, MaxLabelLen, view.Title)
	}
	if !strings.HasSuffix(view.Title, "…") {
		t.Fatalf("truncated title should end in an ellipsis: %q", view.Title)
	}
}

func TestSlugAndProviderLabel(t *testing.T) {
	cases := map[string]string{
		"CC v1":            "cc-v1",
		"  Command Code  ": "command-code",
		"v2":               "v2",
		"":                 "",
		"!!!":              "",
	}
	for in, want := range cases {
		if got := Slug(in); got != want {
			t.Fatalf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
	if got := ProviderLabel(KindCommandCode); got != "CommandCode" {
		t.Fatalf("ProviderLabel = %q", got)
	}
	if got := ProviderLabel("mystery"); got != "mystery" {
		t.Fatalf("unknown ProviderLabel = %q", got)
	}
}

func TestNormalizeRepairsRegistry(t *testing.T) {
	reg := &Registry{
		Items: []Item{
			{ID: "  grok  ", Kind: KindGrok, Label: "  "},
			{ID: "grok", Kind: KindGrok, Label: "dup"},
			{ID: "", Kind: KindGrok, Label: "no id"},
			{ID: "cc", Kind: KindCommandCode},
		},
		Default: "gone",
	}
	reg.Normalize()

	if reg.Version != RegistryVersion {
		t.Fatalf("version = %d", reg.Version)
	}
	if len(reg.Items) != 2 {
		t.Fatalf("items = %+v, want duplicates and blanks dropped", reg.Items)
	}
	if reg.Items[0].Label != "Grok" {
		t.Fatalf("label = %q, want Grok", reg.Items[0].Label)
	}
	if reg.Items[1].Label != "CommandCode" {
		t.Fatalf("label = %q, want CommandCode", reg.Items[1].Label)
	}
	if reg.Default != "grok" {
		t.Fatalf("default = %q, want grok", reg.Default)
	}
}
