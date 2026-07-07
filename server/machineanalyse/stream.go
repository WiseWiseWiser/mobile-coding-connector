package machineanalyse

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/xhd2015/ai-critic/server/streaming/progress"
	"github.com/xhd2015/dot-pkgs/go-pkgs/file/analyse"
)

// AnalyseFilesStream scans HOME and streams one completed entry block at a time.
func AnalyseFilesStream(w http.ResponseWriter, home string) error {
	pw := progress.NewWriter(w)
	if pw == nil {
		return fmt.Errorf("streaming not supported")
	}

	if home == "" {
		home = os.Getenv("HOME")
	}

	resolvedHome, err := analyse.ResolveHome(home)
	if err != nil {
		if emitErr := pw.EmitError(err.Error()); emitErr != nil {
			return emitErr
		}
		return nil
	}

	if err := pw.EmitLog(fmt.Sprintf("home: %s", resolvedHome), true); err != nil {
		return err
	}

	entries, summary, err := analyse.Scan(context.Background(), analyse.Options{
		Home: resolvedHome,
		OnEntry: func(entry analyse.EntryResult) error {
			return pw.EmitLog(analyse.FormatEntryBlock(entry), true)
		},
	})
	if err != nil {
		if emitErr := pw.EmitError(err.Error()); emitErr != nil {
			return emitErr
		}
		return nil
	}

	for _, line := range analyse.FormatSummaryLines(summary) {
		if err := pw.EmitLog(line, true); err != nil {
			return err
		}
	}

	return pw.EmitDone(analyse.SummaryToDone(summary, entries))
}