package agentcli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/ai-critic/macosapp/usageitems"
	"github.com/xhd2015/ai-critic/server/usage"
)

// usageTestEnv runs the real usage item service behind the real HTTP handlers,
// with provider fetching replaced by fixtures.
type usageTestEnv struct {
	svc    *usageitems.Service
	server *httptest.Server
}

type usageFixtureError struct{}

func (usageFixtureError) Error() string { return "Session expired" }

// fixtureSnapshot renders deterministic menu text per kind. A home of
// /tmp/dead stands in for a provider directory with no usable credentials.
func fixtureSnapshot(it usageitems.Item) (usageitems.Snapshot, error) {
	if it.Home == "/tmp/dead" {
		return usageitems.Snapshot{}, usageFixtureError{}
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
			Detail:   "USAGE  GOAT Plan · active\n5% cycle\n5-hour  14% · resets in 2h 4m\nFull breakdown at commandcode.ai/acct/settings/usage",
			UsageURL: "https://commandcode.ai/acct/settings/usage",
		}, nil
	}
}

func newUsageTestEnv(t *testing.T) *usageTestEnv {
	t.Helper()
	active = LocalProfile()

	store := usageitems.NewStore(filepath.Join(t.TempDir(), "usage-items.json"))
	svc := usageitems.NewService(store)
	usageitems.TestExported_SetSnapshotFetcher(svc, fixtureSnapshot)
	if prev := usage.TestExported_SetItemsService(svc); prev != nil {
		t.Cleanup(func() { usage.TestExported_SetItemsService(prev) })
	}

	mux := http.NewServeMux()
	usage.RegisterAPI(mux)
	env := &usageTestEnv{svc: svc, server: httptest.NewServer(mux)}
	t.Cleanup(env.server.Close)
	return env
}

// addItem registers an item without provider validation.
func (e *usageTestEnv) addItem(t *testing.T, item usageitems.Item) {
	t.Helper()
	enabled := true
	if _, _, err := e.svc.Add(usageitems.AddRequest{Item: item, Enabled: &enabled, SkipValidate: true}); err != nil {
		t.Fatalf("seed item %s: %v", item.ID, err)
	}
}

// runCLI executes one usage subcommand and captures stdout and stderr.
func (e *usageTestEnv) runCLI(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	out, errOut, err := captureUsageOutput(t, func() error {
		return runUsage(func() (*client.Client, error) {
			return client.New(e.server.URL, ""), nil
		}, args)
	})
	return out, errOut, err
}

// captureUsageOutput redirects process stdout/stderr while fn runs, because the
// usage command writes through osStdout()/os.Stderr.
func captureUsageOutput(t *testing.T, fn func() error) (string, string, error) {
	t.Helper()
	oldOut, oldErr := os.Stdout, os.Stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stderr: %v", err)
	}
	os.Stdout, os.Stderr = wOut, wErr

	var outBuf, errBuf bytes.Buffer
	done := make(chan struct{}, 2)
	go func() { _, _ = io.Copy(&outBuf, rOut); done <- struct{}{} }()
	go func() { _, _ = io.Copy(&errBuf, rErr); done <- struct{}{} }()

	runErr := fn()

	_ = wOut.Close()
	_ = wErr.Close()
	<-done
	<-done
	os.Stdout, os.Stderr = oldOut, oldErr
	_ = rOut.Close()
	_ = rErr.Close()
	return outBuf.String(), errBuf.String(), runErr
}

func TestRunUsageListShowsItemsAndFooter(t *testing.T) {
	env := newUsageTestEnv(t)
	env.addItem(t, usageitems.Item{ID: "cc-v1", Label: "CC v1", Kind: usageitems.KindCommandCode, Home: "/tmp/cc1"})
	if _, _, err := env.runCLI(t, "add", "--id", "cc-v2", "--kind", "commandcode",
		"--home", "/tmp/cc2", "--disabled", "--no-validate"); err != nil {
		t.Fatalf("usage add: %v", err)
	}

	out, errOut, err := env.runCLI(t, "list")
	if err != nil {
		t.Fatalf("usage list: %v", err)
	}
	if errOut != "" {
		t.Fatalf("stderr = %q, want empty", errOut)
	}
	for _, want := range []string{
		"ID", "DEF", "LABEL", "KIND", "ENABLED", "STATUS", "SUMMARY",
		"cc-v1", "CC v1", "commandcode", "ready",
		"grok", "codex",
		"4 items · default grok · rotate on (60s)",
		"5% used, $66.19 left, 1,870 requests",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("list output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "*") {
		t.Fatalf("there is no default marker while rotating:\n%s", out)
	}
}

func TestRunUsageListMarksPinnedDefault(t *testing.T) {
	env := newUsageTestEnv(t)
	if _, _, err := env.runCLI(t, "default", "codex"); err != nil {
		t.Fatalf("usage default: %v", err)
	}

	out, _, err := env.runCLI(t, "list")
	if err != nil {
		t.Fatalf("usage list: %v", err)
	}
	if !strings.Contains(out, "· default codex · rotate off") {
		t.Fatalf("footer missing:\n%s", out)
	}
	marked := false
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "codex") && strings.Contains(line, "*") {
			marked = true
		}
	}
	if !marked {
		t.Fatalf("the pinned default must be marked:\n%s", out)
	}
}

func TestRunUsageListJSON(t *testing.T) {
	env := newUsageTestEnv(t)
	out, _, err := env.runCLI(t, "list", "--json")
	if err != nil {
		t.Fatalf("usage list --json: %v", err)
	}

	var decoded client.UsageItemsResponse
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("decode json: %v\n%s", err, out)
	}
	if decoded.Default != "grok" || !decoded.Rotate {
		t.Fatalf("selection = %q/%v", decoded.Default, decoded.Rotate)
	}
	if len(decoded.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(decoded.Items))
	}
	for _, item := range decoded.Items {
		if item.Title == "" || item.Dropdown == "" {
			t.Fatalf("item %s missing rendered text: %+v", item.ID, item)
		}
	}
}

func TestRunUsageAddTwoCommandCodeItems(t *testing.T) {
	env := newUsageTestEnv(t)

	out, errOut, err := env.runCLI(t, "add",
		"--id", "cc-v1", "--kind", "commandcode",
		"--label", "CC v1", "--home", "/tmp/commandcode-v1/.commandcode")
	if err != nil {
		t.Fatalf("usage add: %v", err)
	}
	if errOut != "" {
		t.Fatalf("stderr = %q", errOut)
	}
	for _, want := range []string{
		"Added usage item cc-v1 (commandcode)",
		"Label:   CC v1",
		"Home:    /tmp/commandcode-v1/.commandcode",
		"Status:  ready · 5% used, $66.19 left, 1,870 requests, 5h 15%, Weekly 11%, renews in 26d",
		"Show it in the menu bar with: local-agent usage default cc-v1",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("add output missing %q:\n%s", want, out)
		}
	}

	// A second Command Code item with no --label takes the kind's default label.
	out, _, err = env.runCLI(t, "add", "--id", "cc-v2", "--kind", "commandcode",
		"--home", "/tmp/commandcode-v2/.commandcode")
	if err != nil {
		t.Fatalf("usage add: %v", err)
	}
	if !strings.Contains(out, "Label:   CommandCode") {
		t.Fatalf("default label must be the provider name:\n%s", out)
	}

	out, _, err = env.runCLI(t, "list")
	if err != nil {
		t.Fatalf("usage list: %v", err)
	}
	if !strings.Contains(out, "cc-v2") || !strings.Contains(out, "4 items") {
		t.Fatalf("both command code items must be listed:\n%s", out)
	}
	if !strings.Contains(out, "CommandCode  commandcode") {
		t.Fatalf("default-label item must be listed under its provider name:\n%s", out)
	}
}

func TestRunUsageAddValidatesKindAndHome(t *testing.T) {
	env := newUsageTestEnv(t)

	if _, _, err := env.runCLI(t, "add", "--id", "x"); err == nil ||
		!strings.Contains(err.Error(), "--kind is required") {
		t.Fatalf("missing kind error = %v", err)
	}
	if _, _, err := env.runCLI(t, "add", "--kind", "commandcode"); err == nil ||
		!strings.Contains(err.Error(), "--home is required for kind commandcode") {
		t.Fatalf("missing home error = %v", err)
	}
	if _, _, err := env.runCLI(t, "add", "--kind", "nope", "--id", "x"); err == nil ||
		!strings.Contains(err.Error(), "unknown kind") {
		t.Fatalf("unknown kind error = %v", err)
	}
}

func TestRunUsageAddWarnsAndStrictFails(t *testing.T) {
	env := newUsageTestEnv(t)

	out, errOut, err := env.runCLI(t, "add", "--id", "cc-dead", "--kind", "commandcode", "--home", "/tmp/dead")
	if err != nil {
		t.Fatalf("a failed validation must warn, not fail: %v", err)
	}
	if !strings.Contains(errOut, "warning: cc-dead fetch failed: Session expired") {
		t.Fatalf("stderr = %q", errOut)
	}
	if !strings.Contains(out, "Status:  error · Error: Session expired") {
		t.Fatalf("add must show the error status:\n%s", out)
	}

	// --strict turns the same failure into a fatal error and registers nothing.
	_, _, err = env.runCLI(t, "add", "--id", "cc-dead2", "--kind", "commandcode", "--home", "/tmp/dead", "--strict")
	if err == nil || !strings.Contains(err.Error(), "cc-dead2 fetch failed: Session expired") {
		t.Fatalf("strict error = %v", err)
	}
	if _, _, err := env.runCLI(t, "show", "cc-dead2"); err == nil {
		t.Fatal("a strict failure must not register the item")
	}

	if _, errOut, err := env.runCLI(t, "add", "--id", "cc-dead3", "--kind", "commandcode", "--home", "/tmp/dead", "--no-validate"); err != nil {
		t.Fatalf("usage add --no-validate: %v", err)
	} else if errOut != "" {
		t.Fatalf("--no-validate must not warn: %q", errOut)
	}
}

func TestRunUsageUpdateAndRemove(t *testing.T) {
	env := newUsageTestEnv(t)
	if _, _, err := env.runCLI(t, "add", "--id", "cc-v1", "--kind", "commandcode", "--home", "/tmp/cc1", "--no-validate"); err != nil {
		t.Fatalf("usage add: %v", err)
	}

	if _, _, err := env.runCLI(t, "update"); err == nil ||
		!strings.Contains(err.Error(), "requires exactly 1 argument <id>") {
		t.Fatalf("missing id error = %v", err)
	}
	if _, _, err := env.runCLI(t, "update", "cc-v1"); err == nil ||
		!strings.Contains(err.Error(), "no changes specified") {
		t.Fatalf("no-op update error = %v", err)
	}
	if _, _, err := env.runCLI(t, "update", "cc-v1", "--enable", "--disable"); err == nil ||
		!strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("enable+disable error = %v", err)
	}
	if _, _, err := env.runCLI(t, "update", "missing", "--label", "x"); err == nil ||
		!strings.Contains(err.Error(), "unknown usage item") {
		t.Fatalf("unknown item error = %v", err)
	}

	out, _, err := env.runCLI(t, "update", "cc-v1", "--label", "CommandCode v1", "--disable")
	if err != nil {
		t.Fatalf("usage update: %v", err)
	}
	if !strings.Contains(out, "Updated usage item cc-v1") || !strings.Contains(out, "Label:   CommandCode v1") {
		t.Fatalf("update output:\n%s", out)
	}

	// Re-pointing an item at another provider kind re-validates it.
	out, _, err = env.runCLI(t, "update", "cc-v1", "--kind", "codex", "--enable")
	if err != nil {
		t.Fatalf("switching kind must be allowed: %v", err)
	}
	if !strings.Contains(out, "Status:  ready · 38%(Monthly) $12.00/$50.00") {
		t.Fatalf("update must re-render the new provider:\n%s", out)
	}

	out, _, err = env.runCLI(t, "update", "cc-v1", "--kind", "commandcode", "--home", "/tmp/dead")
	if err != nil {
		t.Fatalf("usage update: %v", err)
	}
	if !strings.Contains(out, "Status:  error") {
		t.Fatalf("update must show the new error status:\n%s", out)
	}

	if _, _, err := env.runCLI(t, "default", "codex"); err != nil {
		t.Fatalf("usage default: %v", err)
	}
	out, errOut, err := env.runCLI(t, "remove", "cc-v1")
	if err != nil {
		t.Fatalf("usage remove: %v", err)
	}
	if !strings.Contains(out, "Removed usage item cc-v1") {
		t.Fatalf("remove output:\n%s", out)
	}
	if errOut != "" {
		t.Fatalf("removing a non-default item must not warn: %q", errOut)
	}
	if _, _, err := env.runCLI(t, "remove", "cc-v1"); err == nil ||
		!strings.Contains(err.Error(), "unknown usage item") {
		t.Fatalf("removing twice error = %v", err)
	}
}

func TestRunUsageRemovePinnedDefaultWarns(t *testing.T) {
	env := newUsageTestEnv(t)
	if _, _, err := env.runCLI(t, "default", "codex"); err != nil {
		t.Fatalf("usage default: %v", err)
	}

	out, errOut, err := env.runCLI(t, "remove", "codex")
	if err != nil {
		t.Fatalf("usage remove: %v", err)
	}
	if !strings.Contains(errOut, "warning: codex was the menu bar default; default is now grok") {
		t.Fatalf("stderr = %q", errOut)
	}
	if !strings.Contains(out, "Removed usage item codex") {
		t.Fatalf("stdout = %q", out)
	}
}

func TestRunUsageDefaultPinsAndRotates(t *testing.T) {
	env := newUsageTestEnv(t)

	out, _, err := env.runCLI(t, "default", "codex")
	if err != nil {
		t.Fatalf("usage default: %v", err)
	}
	if !strings.Contains(out, "Menu bar default: grok → codex (rotate off)") {
		t.Fatalf("pin output:\n%s", out)
	}

	out, _, err = env.runCLI(t, "default", "codex")
	if err != nil {
		t.Fatalf("usage default: %v", err)
	}
	if !strings.Contains(out, "Menu bar default: codex (rotate off)") {
		t.Fatalf("re-pin output:\n%s", out)
	}

	out, _, err = env.runCLI(t, "default", "--rotate", "--start", "grok")
	if err != nil {
		t.Fatalf("usage default --rotate: %v", err)
	}
	if !strings.Contains(out, "Menu bar default: rotating over 2 enabled items (60s), starting at grok") {
		t.Fatalf("rotate output:\n%s", out)
	}

	if _, _, err := env.runCLI(t, "default"); err == nil ||
		!strings.Contains(err.Error(), "pass <id>, or --rotate") {
		t.Fatalf("bare default error = %v", err)
	}
	if _, _, err := env.runCLI(t, "default", "missing"); err == nil ||
		!strings.Contains(err.Error(), "unknown usage item") {
		t.Fatalf("unknown default error = %v", err)
	}
	if _, _, err := env.runCLI(t, "default", "grok", "--start", "codex"); err == nil ||
		!strings.Contains(err.Error(), "disagree") {
		t.Fatalf("conflicting id error = %v", err)
	}
}

func TestRunUsageShowPanelLinesAndJSON(t *testing.T) {
	env := newUsageTestEnv(t)
	if _, _, err := env.runCLI(t, "add", "--id", "cc-v1", "--kind", "commandcode", "--label", "CC v1", "--home", "/tmp/cc1"); err != nil {
		t.Fatalf("usage add: %v", err)
	}

	out, _, err := env.runCLI(t, "show", "cc-v1")
	if err != nil {
		t.Fatalf("usage show: %v", err)
	}
	for _, want := range []string{"USAGE  GOAT Plan · active", "5-hour  14% · resets in 2h 4m", "Full breakdown at commandcode.ai"} {
		if !strings.Contains(out, want) {
			t.Fatalf("show must print the provider panel, missing %q:\n%s", want, out)
		}
	}

	out, _, err = env.runCLI(t, "show")
	if err != nil {
		t.Fatalf("usage show: %v", err)
	}
	var lines []string
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) != 3 {
		t.Fatalf("show without an id must print one line per item, got %d:\n%s", len(lines), out)
	}
	if lines[0] != "Grok: 61%(Weekly), Reset July 17, 08:55, left 4d" {
		t.Fatalf("grok line = %q", lines[0])
	}
	if lines[1] != "Codex: 38%(Monthly) $12.00/$50.00, Reset Aug 1, 09:00, left 12d" {
		t.Fatalf("codex line = %q", lines[1])
	}
	if lines[2] != "CC v1: 5% used, $66.19 left, 1,870 requests, 5h 15%, Weekly 11%, renews in 26d" {
		t.Fatalf("command code line = %q", lines[2])
	}

	if _, _, err := env.runCLI(t, "show", "missing"); err == nil ||
		!strings.Contains(err.Error(), `unknown usage item "missing"`) {
		t.Fatalf("unknown show error = %v", err)
	}

	// A disabled item is out of the menu bar and never fetched, but still listed.
	if _, _, err := env.runCLI(t, "add", "--id", "cc-v2", "--kind", "commandcode",
		"--label", "CC v2", "--home", "/tmp/cc2", "--disabled", "--no-validate"); err != nil {
		t.Fatalf("usage add: %v", err)
	}
	out, _, err = env.runCLI(t, "show")
	if err != nil {
		t.Fatalf("usage show: %v", err)
	}
	if !strings.Contains(out, "CC v2: Loading...") {
		t.Fatalf("show must include disabled items, unfetched:\n%s", out)
	}
	if lines := strings.Count(strings.TrimSpace(out), "\n\n") + 1; lines != 4 {
		t.Fatalf("show printed %d lines, want 4:\n%s", lines, out)
	}

	out, _, err = env.runCLI(t, "show", "cc-v1", "--json")
	if err != nil {
		t.Fatalf("usage show --json: %v", err)
	}
	var item client.UsageItemView
	if err := json.Unmarshal([]byte(out), &item); err != nil {
		t.Fatalf("decode json: %v\n%s", err, out)
	}
	if item.ID != "cc-v1" || item.UsageURL != "https://commandcode.ai/acct/settings/usage" {
		t.Fatalf("json item = %+v", item)
	}
}

func TestRunUsageHelpAndUnknownSubcommand(t *testing.T) {
	env := newUsageTestEnv(t)

	out, _, err := env.runCLI(t)
	if err != nil {
		t.Fatalf("bare usage: %v", err)
	}
	for _, want := range []string{
		"Usage: local-agent usage <subcommand>",
		"list", "add", "update", "remove", "default", "show",
		"Kinds: grok, codex, commandcode (commandcode requires --home).",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("help missing %q:\n%s", want, out)
		}
	}

	if _, _, err := env.runCLI(t, "bogus"); err == nil ||
		!strings.Contains(err.Error(), "unknown usage subcommand: bogus") {
		t.Fatalf("unknown subcommand error = %v", err)
	}
}

func TestRunUsageAddDuplicateReportsCleanError(t *testing.T) {
	env := newUsageTestEnv(t)

	_, _, err := env.runCLI(t, "add", "--id", "grok", "--kind", "grok")
	if err == nil {
		t.Fatal("adding a duplicate id must fail")
	}
	if got := err.Error(); !strings.Contains(got, `usage item "grok" already exists`) {
		t.Fatalf("error = %q", got)
	}
	if strings.Contains(err.Error(), "409") {
		t.Fatalf("the CLI must strip the HTTP status prefix: %q", err.Error())
	}
}

// countingFetch returns a fresh ready snapshot per call so tests can tell a
// fresh fetch from a cached render.
func countingFetch(count *int) func(usageitems.Item) (usageitems.Snapshot, error) {
	return func(usageitems.Item) (usageitems.Snapshot, error) {
		*count++
		return usageitems.Snapshot{
			Status:    usageitems.StatusReady,
			Percent:   "5%",
			Body:      fmt.Sprintf("5%% used, fetch #%d", *count),
			UpdatedAt: "2026-09-15T12:00:00Z",
		}, nil
	}
}

func TestRunUsageShowFreshByDefaultCachedOnFlag(t *testing.T) {
	env := newUsageTestEnv(t)
	var count int
	usageitems.TestExported_SetSnapshotFetcher(env.svc, countingFetch(&count))

	// Default: each show fetches fresh from the provider.
	if _, _, err := env.runCLI(t, "show"); err != nil {
		t.Fatalf("usage show: %v", err)
	}
	if count != 2 {
		t.Fatalf("fresh show fetched %d times, want 2 (grok + codex)", count)
	}

	// --cached: no additional fetches, prints the same cached snapshots.
	out, _, err := env.runCLI(t, "show", "--cached")
	if err != nil {
		t.Fatalf("usage show --cached: %v", err)
	}
	if count != 2 {
		t.Fatalf("cached show fetched %d times, want still 2", count)
	}
	for _, want := range []string{"Grok: 5% used, fetch #1", "Codex: 5% used, fetch #2"} {
		if !strings.Contains(out, want) {
			t.Fatalf("cached show missing %q:\n%s", want, out)
		}
	}

	// Fresh again: fetches once more per enabled item.
	out, _, err = env.runCLI(t, "show")
	if err != nil {
		t.Fatalf("usage show: %v", err)
	}
	if count != 4 {
		t.Fatalf("second fresh show fetched %d times, want 4", count)
	}
	if !strings.Contains(out, "fetch #3") || !strings.Contains(out, "fetch #4") {
		t.Fatalf("fresh show must reflect new fetches:\n%s", out)
	}
}

func TestRunUsageListFreshByDefaultCachedOnFlag(t *testing.T) {
	env := newUsageTestEnv(t)
	var count int
	usageitems.TestExported_SetSnapshotFetcher(env.svc, countingFetch(&count))

	if _, _, err := env.runCLI(t, "list"); err != nil {
		t.Fatalf("usage list: %v", err)
	}
	if count != 2 {
		t.Fatalf("fresh list fetched %d times, want 2", count)
	}
	if _, _, err := env.runCLI(t, "list", "--cached"); err != nil {
		t.Fatalf("usage list --cached: %v", err)
	}
	if count != 2 {
		t.Fatalf("cached list fetched %d times, want still 2", count)
	}
}
