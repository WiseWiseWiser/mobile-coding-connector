package usage

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/xhd2015/ai-critic/macosapp/usageitems"
)

var errMissingID = errors.New("usage item: --id is required")

func decodeUsageJSON(r *http.Request, out any) error {
	if err := json.NewDecoder(r.Body).Decode(out); err != nil {
		return fmt.Errorf("invalid json: %v", err)
	}
	return nil
}

func writeUsageJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeUsageError(w http.ResponseWriter, status int, err error) {
	writeUsageJSON(w, status, map[string]string{"error": err.Error()})
}

// usageStatusForError maps item errors to HTTP status codes.
func usageStatusForError(err error) int {
	var itemErr *usageitems.ItemError
	if errors.As(err, &itemErr) {
		switch itemErr.Kind {
		case usageitems.ErrorNotFound:
			return http.StatusNotFound
		case usageitems.ErrorConflict:
			return http.StatusConflict
		default:
			return http.StatusBadRequest
		}
	}
	return http.StatusInternalServerError
}
