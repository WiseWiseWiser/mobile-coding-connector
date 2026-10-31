package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const tabCloseConfirmationFeature = "tab-close-confirm-v1"

// CloseTabConfirmation is an ephemeral, extension-owned close operation.
type CloseTabConfirmation struct {
	OperationID string `json:"operation_id"`
	TabID       int64  `json:"tab_id"`
}

type remoteTabCloseResult struct {
	Status      string `json:"status"`
	OperationID string `json:"operation_id"`
	TabID       int64  `json:"tab_id"`
}

func (m *Manager) remoteTabCloseJob(ctx context.Context, sessionID, method string, params map[string]any) (*remoteTabCloseResult, error) {
	data, err := m.jobIn(ctx, sessionID, 0, "cdp", map[string]any{"method": method, "params": params}, 35*time.Second)
	if err != nil {
		return nil, err
	}
	var response struct {
		Result remoteTabCloseResult `json:"result"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("decode tab close result: %w", err)
	}
	return &response.Result, nil
}

func isTabCloseTimeout(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || strings.Contains(strings.ToLower(err.Error()), "timeout")
}

func unknownTabCloseError() error {
	return errors.New("tab close outcome is unknown; check Chrome for a dialog and refresh the tab list before retrying")
}

// ResolveTabClose answers only the extension-owned operation in the given session.
// It never retries a close and never infers that an acknowledged command closed a tab.
func (m *Manager) ResolveTabClose(ctx context.Context, sessionID, operationID string, accept bool) (*sessionInfo, error) {
	if m.isReadOnly() {
		return nil, errReadOnly
	}
	if sessionID == "" {
		sessionID = m.sessionIDValue()
	}
	if strings.TrimSpace(operationID) == "" {
		return nil, errors.New("missing close confirmation operation id")
	}
	result, err := m.remoteTabCloseJob(ctx, sessionID, "BrowserAgent.resolveCloseTab", map[string]any{"operation_id": operationID, "accept": accept})
	if err != nil {
		return nil, fmt.Errorf("resolve tab close: %w", err)
	}
	var info *sessionInfo
	switch result.Status {
	case "closed":
		info, err = m.retarget(ctx, sessionID, 0)
	case "kept":
		info, err = m.fetchInfo(ctx, sessionID)
	default:
		return nil, unknownTabCloseError()
	}
	if err != nil {
		return nil, err
	}
	info.CloseOutcome = result.Status
	return info, nil
}
