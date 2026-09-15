# Usage Item Registry, Rendering, and API Doctests

Tests for `macosapp/usageitems` — the registry that decides what the macOS menu
bar shows — plus the `server/usage` handlers the `local-agent usage` CLI drives.

# DSN (Domain Specific Notion)

**Participants**

- **Registry file (`~/.ai-critic/usage-items.json`)** — the persisted item list
  (`id`, `label`, `kind`, `enabled`, `home`, `api_url`) plus the menu-bar
  selection (`default`, `rotate`). Written by the server, edited only through the
  `usage` CLI.
- **Usage item service (`macosapp/usageitems`)** — loads the registry, seeds
  grok + codex when no file exists, keeps one fetch worker per item, and renders
  each item's menu-bar text.
- **Item kinds** — `grok` and `codex` read the machine's default account;
  `commandcode` reads the sandbox directory named by `home` (required).
- **Rendered view** — `title` (menu-bar text, capped at 40 runes) and `dropdown`
  (one line per item). The macOS app prints both verbatim, so all formatting
  lives here.
- **Status** — `loading`, `ready`, or `error`. Ready titles are
  `{label} {percent}`, error titles `{label} err`, loading titles `{label} ...`.
- **Validation** — adding or re-pointing an item fetches the provider once.
  Failure warns by default (the CLI prints `warning:` on stderr and exits 0) and
  rejects with `--strict`.
- **Selection** — `default` pins the menu-bar item; `rotate` cycles the enabled
  items in registry order. `usage default <id>` turns rotation off.
- **HTTP surface (`server/usage`)** — `GET /api/usage/items` (public) plus
  authenticated `add` / `update` / `remove` / `default`. Status codes:
  400 invalid, 404 not found, 409 conflict, 500 otherwise.

**Behaviors**

- First run seeds `grok` and `codex`, labeled `Grok` / `Codex`, default `grok`,
  rotation on — today's menu bar.
- An omitted `--label` becomes the provider name (`CommandCode`), suffixed with
  the id once the name is taken (`CommandCode cc-v2`).
- An omitted `--id` becomes the slug of the label (`CC v2` → `cc-v2`).
- A `commandcode` item without `home` is invalid, on both the CLI and the API.
- Disabled items stay in the registry, are excluded from rotation, and are never
  fetched (they render as `loading`).
- Removing the pinned default reassigns it to the first enabled item and reports
  the change as a warning.
- The menu-bar title for a `commandcode` item is the cycle credit percent
  (`CommandCode 5%`).

## Version

0.1.0

## Decision Tree

```
[usage items]
 |
 +-- registry/                        (GROUP)  persisted item list
 |    +-- seeds-grok-and-codex/       (LEAF)   first run: grok + codex, rotating
 |
 +-- add/                             (GROUP)  registration + validation
 |    +-- default-label-suffix/       (LEAF)   CommandCode, then CommandCode cc-v2
 |    +-- derives-id-from-label/      (LEAF)   "CC v2" -> id cc-v2
 |    +-- warns-when-provider-fails/  (LEAF)   warning + registered error item
 |    +-- strict-rejects-provider-error/ (LEAF) --strict rejects, registers nothing
 |    +-- rejects-missing-home/       (LEAF)   commandcode requires --home
 |
 +-- render/                          (GROUP)  menu-bar text
 |    +-- two-commandcode-items/      (LEAF)   v1 and v2 side by side
 |    +-- rotation-order/             (LEAF)   enabled order, disabled skipped
 |
 +-- default/                         (GROUP)  menu-bar selection
 |    +-- pin-stops-rotation/         (LEAF)   pinned item + persisted choice
 |
 +-- remove/                          (GROUP)  deletion
 |    +-- pinned-default-falls-back/  (LEAF)   fallback default + warning
 |
 +-- api/                             (GROUP)  HTTP surface and status codes
      +-- list-returns-rendered-items/ (LEAF)  GET /api/usage/items payload
      +-- add-conflict-409/            (LEAF)  duplicate id
      +-- unknown-item-404/            (LEAF)  unknown id
      +-- invalid-item-400/            (LEAF)  commandcode without home
      +-- missing-id-400/              (LEAF)  remove without id
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `registry/seeds-grok-and-codex` | Missing file seeds grok + codex, default grok, rotating |
| 2 | `add/default-label-suffix` | Omitting `--label` twice yields `CommandCode`, `CommandCode cc-v2` |
| 3 | `add/derives-id-from-label` | `CC v2` gets id `cc-v2` and keeps its label |
| 4 | `add/warns-when-provider-fails` | Failed validation warns, item registered with error text |
| 5 | `add/strict-rejects-provider-error` | `--strict` rejects and registers nothing |
| 6 | `add/rejects-missing-home` | `commandcode` without home is invalid |
| 7 | `render/two-commandcode-items` | Both Command Code items render distinct title/dropdown |
| 8 | `render/rotation-order` | Registry order kept; disabled item never fetched |
| 9 | `default/pin-stops-rotation` | Pinning persists `default` + `rotate=false` |
| 10 | `remove/pinned-default-falls-back` | Removing the default falls back to grok with a warning |
| 11 | `api/list-returns-rendered-items` | GET payload carries rendered text and selection |
| 12 | `api/add-conflict-409` | Duplicate id → 409 |
| 13 | `api/unknown-item-404` | Unknown id → 404 with known ids |
| 14 | `api/invalid-item-400` | Missing home → 400 |
| 15 | `api/missing-id-400` | Remove without id → 400 |

## Parameter Coverage

| Leaf | Op | Seed | FailHome | Expect |
|------|-----|------|----------|--------|
| seeds-grok-and-codex | registry | (none → seeded) | — | 2 items, default grok |
| default-label-suffix | add | (none) | — | 2 added, no warnings |
| derives-id-from-label | add | (none) | — | id derived from label |
| warns-when-provider-fails | add | (none) | /tmp/dead | 1 warning, item registered |
| strict-rejects-provider-error | add | (none) | /tmp/dead | invalid, nothing added |
| rejects-missing-home | add | (none) | — | invalid |
| two-commandcode-items | list | grok+codex+cc-v1+cc-v2 | — | both ready, pinned cc-v1 |
| rotation-order | list | codex disabled | — | codex loading, rotate on |
| pin-stops-rotation | default | grok+codex+cc-v1+cc-v2 | — | default cc-v2, rotate off |
| pinned-default-falls-back | remove | pinned cc-v1 | — | default grok + warning |
| list-returns-rendered-items | api | (none → seeded) | — | 200, title/dropdown keys |
| add-conflict-409 | api | (none → seeded) | — | 409 |
| unknown-item-404 | api | (none → seeded) | — | 404 |
| invalid-item-400 | api | (none → seeded) | — | 400 |
| missing-id-400 | api | (none → seeded) | — | 400 |

## How to Run

```sh
doctest vet ./tests/usage-items
doctest test ./tests/usage-items/...
```

```go
import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/macosapp/usageitems"
	"github.com/xhd2015/ai-critic/server/usage"
	"github.com/xhd2015/doctest/session"
)

type Request struct {
	Op string

	// Seed registry contents. Empty SeedItems keeps the built-in grok+codex seed.
	SeedItems   []usageitems.Item
	SeedDefault string
	SeedRotate  bool

	// Provider fixture: an item whose Home is FailHome fails to fetch.
	FailHome string

	// op=add: items registered in order.
	AddItems        []usageitems.Item
	AddEnabled      *bool
	AddDefault      bool
	AddStrict       bool
	AddSkipValidate bool

	// op=update
	TargetID     string
	SetLabel     string
	SetKind      string
	SetHome      string
	SetEnabled   *bool
	SkipValidate bool
	Strict       bool

	// op=default / op=remove
	Rotate bool

	// op=api
	APIMethod string
	APIPath   string
	APIBody   string
}

type Response struct {
	Item     *usageitems.ItemView
	Items    []usageitems.ItemView
	Added    []usageitems.ItemView
	Default  string
	Rotate   bool
	Warnings []string
	Removed  string
	Err      string
	ErrKind  string
	Registry string

	APIStatus int
	APIBody   string
	APIError  string
}

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	return nil
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	resp := &Response{}
	store := usageitems.NewStore(filepath.Join(t.TempDir(), "usage-items.json"))
	resp.Registry = store.Path()

	if len(req.SeedItems) > 0 {
		reg := &usageitems.Registry{
			Version: usageitems.RegistryVersion,
			Default: req.SeedDefault,
			Rotate:  req.SeedRotate,
			Items:   req.SeedItems,
		}
		if err := store.Save(reg); err != nil {
			return nil, err
		}
	}

	svc := usageitems.NewService(store)
	usageitems.TestExported_SetSnapshotFetcher(svc, func(it usageitems.Item) (usageitems.Snapshot, error) {
		return fixtureSnapshot(it, req.FailHome)
	})
	prev := usage.TestExported_SetItemsService(svc)
	t.Cleanup(func() { usage.TestExported_SetItemsService(prev) })

	switch req.Op {
	case "registry":
		reg, err := svc.Registry()
		if err != nil {
			return nil, err
		}
		resp.Default = reg.Default
		resp.Rotate = reg.Rotate
		for _, it := range reg.Items {
			resp.Items = append(resp.Items, usageitems.ItemView{Item: it})
		}
		return resp, nil

	case "list":
		list, err := svc.List()
		if err != nil {
			return nil, err
		}
		resp.Items = list.Items
		resp.Default = list.Default
		resp.Rotate = list.Rotate
		return resp, nil

	case "show":
		view, err := svc.Show(req.TargetID)
		if recordItemErr(resp, err) {
			return resp, nil
		}
		resp.Item = view
		resp.Default = view.Label
		return resp, nil

	case "add":
		return runAdd(svc, req, resp)

	case "update":
		var upd usageitems.ItemUpdate
		if req.SetLabel != "" {
			upd.Label = &req.SetLabel
		}
		if req.SetKind != "" {
			upd.Kind = &req.SetKind
		}
		if req.SetHome != "" {
			upd.Home = &req.SetHome
		}
		if req.SetEnabled != nil {
			upd.Enabled = req.SetEnabled
		}
		view, warnings, err := svc.Update(usageitems.UpdateRequest{
			ID:           req.TargetID,
			Update:       upd,
			SkipValidate: req.SkipValidate,
			Strict:       req.Strict,
		})
		resp.Warnings = warnings
		if recordItemErr(resp, err) {
			return resp, nil
		}
		resp.Item = view
		return resp, nil

	case "remove":
		result, err := svc.Remove(req.TargetID)
		if recordItemErr(resp, err) {
			return resp, nil
		}
		resp.Removed = result.Removed
		resp.Default = result.Default
		resp.Warnings = result.Warnings
		return resp, nil

	case "default":
		list, err := svc.SetDefault(req.TargetID, req.Rotate)
		if recordItemErr(resp, err) {
			return resp, nil
		}
		resp.Items = list.Items
		resp.Default = list.Default
		resp.Rotate = list.Rotate
		return resp, nil

	case "api":
		return runAPI(t, req, resp)

	default:
		return nil, fmt.Errorf("unknown op %q", req.Op)
	}
}

func runAdd(svc *usageitems.Service, req *Request, resp *Response) (*Response, error) {
	for _, item := range req.AddItems {
		view, warnings, err := svc.Add(usageitems.AddRequest{
			Item:         item,
			Enabled:      req.AddEnabled,
			Default:      req.AddDefault,
			SkipValidate: req.AddSkipValidate,
			Strict:       req.AddStrict,
		})
		resp.Warnings = append(resp.Warnings, warnings...)
		if recordItemErr(resp, err) {
			return resp, nil
		}
		resp.Added = append(resp.Added, *view)
	}
	if len(resp.Added) > 0 {
		last := resp.Added[len(resp.Added)-1]
		resp.Item = &last
		resp.Default = last.ID
	}
	return resp, nil
}

// runAPI drives the real handlers over HTTP so status mapping is exercised.
func runAPI(t *testing.T, req *Request, resp *Response) (*Response, error) {
	t.Helper()
	mux := http.NewServeMux()
	usage.RegisterAPI(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	method := req.APIMethod
	if method == "" {
		method = http.MethodGet
	}
	var body io.Reader
	if strings.TrimSpace(req.APIBody) != "" {
		body = strings.NewReader(req.APIBody)
	}
	httpReq, err := http.NewRequest(method, server.URL+req.APIPath, body)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()
	raw, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}
	resp.APIStatus = httpResp.StatusCode
	resp.APIBody = string(raw)

	var list usageitems.ListResponse
	if json.Unmarshal(raw, &list) == nil && len(list.Items) > 0 {
		resp.Items = list.Items
		resp.Default = list.Default
		resp.Rotate = list.Rotate
	}
	var failure struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(raw, &failure) == nil && failure.Error != "" {
		resp.APIError = failure.Error
	}
	return resp, nil
}

// fixtureSnapshot renders provider text per kind; FailHome stands in for a
// provider directory without usable credentials.
func fixtureSnapshot(it usageitems.Item, failHome string) (usageitems.Snapshot, error) {
	if failHome != "" && it.Home == failHome {
		return usageitems.Snapshot{}, errors.New("Session expired")
	}
	switch it.Kind {
	case usageitems.KindGrok:
		return usageitems.Snapshot{
			Status:   usageitems.StatusReady,
			Percent:  "61%",
			Body:     "61%(Weekly), Reset July 17, 08:55, left 4d",
			UsageURL: "https://grok.example/usage",
		}, nil
	case usageitems.KindCodex:
		return usageitems.Snapshot{
			Status:  usageitems.StatusReady,
			Percent: "38%",
			Body:    "38%(Monthly) $12.00/$50.00, Reset Aug 1, 09:00, left 12d",
		}, nil
	default:
		return usageitems.Snapshot{
			Status:   usageitems.StatusReady,
			Percent:  "5%",
			Body:     "5% used, $66.19 left, 1,870 requests, 5h 15%, Weekly 11%, renews in 26d",
			Detail:   "USAGE  GOAT Plan · active\n5% cycle\nFull breakdown at commandcode.ai/acct/settings/usage",
			UsageURL: "https://commandcode.ai/acct/settings/usage",
		}, nil
	}
}

// recordItemErr stores a user-facing item error instead of failing the leaf.
func recordItemErr(resp *Response, err error) bool {
	if err == nil {
		return false
	}
	resp.Err = err.Error()
	var itemErr *usageitems.ItemError
	if errors.As(err, &itemErr) {
		resp.ErrKind = string(itemErr.Kind)
	}
	return true
}

// usageItem builds an enabled registry entry.
func usageItem(id, label, kind, home string) usageitems.Item {
	return usageitems.Item{ID: id, Label: label, Kind: kind, Home: home, Enabled: true}
}

// commandCodeItems is the two sandbox Command Code entries used by most leaves.
func commandCodeItems() []usageitems.Item {
	return []usageitems.Item{
		usageItem("cc-v1", "CC v1", usageitems.KindCommandCode, "/tmp/commandcode-v1/.commandcode"),
		usageItem("cc-v2", "CC v2", usageitems.KindCommandCode, "/tmp/commandcode-v2/.commandcode"),
	}
}

// grokCodexItems is the built-in seed written out as explicit entries.
func grokCodexItems() []usageitems.Item {
	return []usageitems.Item{
		usageItem("grok", "Grok", usageitems.KindGrok, ""),
		usageItem("codex", "Codex", usageitems.KindCodex, ""),
	}
}

// readRegistry reads the persisted registry, so leaves can assert what the
// server wrote to disk.
func readRegistry(t *testing.T, path string) usageitems.Registry {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read registry %s: %v", path, err)
	}
	var reg usageitems.Registry
	if err := json.Unmarshal(raw, &reg); err != nil {
		t.Fatalf("parse registry %s: %v", path, err)
	}
	return reg
}

// viewsByID indexes rendered items for leaf assertions.
func viewsByID(items []usageitems.ItemView) map[string]usageitems.ItemView {
	out := make(map[string]usageitems.ItemView, len(items))
	for _, item := range items {
		out[item.ID] = item
	}
	return out
}
```
