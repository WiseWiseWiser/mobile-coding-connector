package streamcmd

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/xhd2015/ai-critic/client"
)

// PrintFlags selects builtin event printers (A-path defaults).
type PrintFlags uint

const (
	Logs PrintFlags = 1 << iota
	ProgressChecks
	Sections
	Meta
)

// EventHandler handles one stream event.
type EventHandler func(ev client.StreamEvent) error

// Printer holds optional per-type overrides (B-path).
type Printer struct {
	Log      EventHandler
	Progress EventHandler
	Section  EventHandler
	Meta     EventHandler
	Done     EventHandler
	Error    EventHandler
}

// Spec describes one streaming CLI command.
type Spec struct {
	Method  string
	Path    string
	Query   url.Values
	Body    any
	Print   PrintFlags
	Printer Printer
	After   func(done map[string]any) error
}

// Run streams from the server and prints events incrementally.
func Run(getClient func() (*client.Client, error), spec Spec) error {
	c, err := getClient()
	if err != nil {
		return err
	}

	path := spec.Path
	if len(spec.Query) > 0 {
		path += "?" + spec.Query.Encode()
	}

	handlers := buildHandlers(spec)

	var donePayload map[string]any
	_, err = c.ConsumeStream(spec.Method, path, spec.Body, func(ev client.StreamEvent, raw map[string]any) error {
		switch ev.Type {
		case "error":
			if spec.Printer.Error != nil {
				return spec.Printer.Error(ev)
			}
			msg := ev.Message
			if msg == "" {
				msg = "stream failed"
			}
			return errors.New(msg)
		case "done":
			donePayload = raw
			if handlers.Done != nil {
				return handlers.Done(ev)
			}
			return nil
		}

		handler := handlers.forType(ev.Type)
		if handler == nil {
			return nil
		}
		return handler(ev)
	})
	if err != nil {
		return err
	}

	if spec.After != nil {
		return spec.After(donePayload)
	}
	return nil
}

type handlerTable struct {
	Log      EventHandler
	Progress EventHandler
	Section  EventHandler
	Meta     EventHandler
	Done     EventHandler
}

func (h handlerTable) forType(typ string) EventHandler {
	switch typ {
	case "log":
		return h.Log
	case "progress":
		return h.Progress
	case "section":
		return h.Section
	case "meta":
		return h.Meta
	default:
		return nil
	}
}

func buildHandlers(spec Spec) handlerTable {
	var table handlerTable

	if spec.Print&Logs != 0 {
		table.Log = DefaultLog
	}
	if spec.Print&ProgressChecks != 0 {
		table.Progress = DefaultProgress
	}
	if spec.Print&Sections != 0 {
		table.Section = DefaultSection
	}
	if spec.Print&Meta != 0 {
		table.Meta = DefaultMeta
	}

	if spec.Printer.Log != nil {
		table.Log = spec.Printer.Log
	}
	if spec.Printer.Progress != nil {
		table.Progress = spec.Printer.Progress
	}
	if spec.Printer.Section != nil {
		table.Section = spec.Printer.Section
	}
	if spec.Printer.Meta != nil {
		table.Meta = spec.Printer.Meta
	}
	table.Done = spec.Printer.Done

	return table
}

// MethodGET is a convenience constant for GET streaming endpoints.
const MethodGET = http.MethodGet