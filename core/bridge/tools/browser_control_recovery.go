package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

type browserActionKind string

const browserSessionInvalidErrorCode = "browser_session_invalid"

const (
	browserActionGoto       browserActionKind = "goto"
	browserActionClick      browserActionKind = "click"
	browserActionType       browserActionKind = "type"
	browserActionPress      browserActionKind = "press"
	browserActionEvaluate   browserActionKind = "evaluate"
	browserActionContent    browserActionKind = "content"
	browserActionScreenshot browserActionKind = "screenshot"
	browserActionInfo       browserActionKind = "info"
)

type browserRecoveryMeta struct {
	Recovered bool
	Reason    string
}

func (t *BrowserControlTool) runWithRecovery(
	ctx context.Context,
	action browserActionKind,
	session *browserSession,
	preflightTimeout time.Duration,
	run func(*browserSession) error,
) (browserRecoveryMeta, error) {
	if err := t.preflightBrowserSession(session, preflightTimeout); err != nil {
		return browserRecoveryMeta{}, err
	}
	runErr := run(session)
	if runErr == nil {
		return browserRecoveryMeta{}, nil
	}
	if !isBrowserSessionInvalidError(runErr) {
		return browserRecoveryMeta{}, runErr
	}
	if !browserAutoRecoveryAllowed(action) {
		return browserRecoveryMeta{}, wrapBrowserSessionInvalid(runErr)
	}
	reason := browserSessionInvalidReason(runErr)
	recoveredSession, err := t.recoverSession(ctx, session, preflightTimeout)
	if err != nil {
		return browserRecoveryMeta{}, fmt.Errorf("%w; recovery failed: %v", wrapBrowserSessionInvalid(runErr), err)
	}
	if err := t.preflightBrowserSession(recoveredSession, preflightTimeout); err != nil {
		return browserRecoveryMeta{}, fmt.Errorf("%w; recovery preflight failed: %v", wrapBrowserSessionInvalid(runErr), err)
	}
	if err := t.repairRecoveredSessionContext(recoveredSession, action, preflightTimeout); err != nil {
		return browserRecoveryMeta{}, fmt.Errorf("%w; recovery context repair failed: %v", wrapBrowserSessionInvalid(runErr), err)
	}
	if rerunErr := run(recoveredSession); rerunErr != nil {
		if isBrowserSessionInvalidError(rerunErr) {
			return browserRecoveryMeta{}, wrapBrowserSessionInvalid(rerunErr)
		}
		return browserRecoveryMeta{}, rerunErr
	}
	return browserRecoveryMeta{Recovered: true, Reason: reason}, nil
}

func (t *BrowserControlTool) recoverSession(
	ctx context.Context,
	session *browserSession,
	timeout time.Duration,
) (*browserSession, error) {
	if session == nil {
		return nil, fmt.Errorf("session is required")
	}
	scopeKey := t.sessionScope(ctx)
	if scopeKey == browserSessionGlobalScope && strings.TrimSpace(session.scopeKey) != "" {
		scopeKey = strings.TrimSpace(session.scopeKey)
	}
	if scopeKey == "" {
		scopeKey = browserSessionGlobalScope
	}
	if strings.TrimSpace(session.wsEndpoint) == "" {
		return nil, fmt.Errorf("ws endpoint is empty")
	}
	recoveredEndpoint, err := recoverBrowserWSEndpoint(session.wsEndpoint, timeout)
	if err != nil {
		return nil, err
	}
	if err := t.replaceSession(scopeKey, session.id, recoveredEndpoint); err != nil {
		return nil, err
	}
	recovered, err := t.sessionByID(scopeKey, session.id)
	if err != nil {
		return nil, err
	}
	recovered.copyPageStateFrom(session)
	t.ensureSessionPool().setActive(scopeKey, recovered.id)
	return recovered, nil
}

func recoverBrowserWSEndpoint(endpoint string, timeout time.Duration) (string, error) {
	discovery, err := discoverBrowserEndpoint(endpoint, timeout)
	if err != nil {
		return "", err
	}
	return discovery.WSEndpoint, nil
}

func (t *BrowserControlTool) repairRecoveredSessionContext(
	session *browserSession,
	action browserActionKind,
	timeout time.Duration,
) error {
	if !browserContextRepairRequired(action) {
		return nil
	}
	targetURL, _ := session.pageState()
	if targetURL == "" {
		return nil
	}
	currentURL, err := browserSessionLocation(session, timeout)
	if !browserShouldRepairSessionByGoto(currentURL, err) {
		if err != nil {
			return err
		}
		return nil
	}
	return browserRepairSessionByGoto(session, targetURL, timeout)
}

func browserShouldRepairSessionByGoto(currentURL string, err error) bool {
	if err != nil {
		return isBrowserSessionInvalidError(err)
	}
	return isBrowserBlankURL(currentURL)
}

func browserSessionLocation(session *browserSession, timeout time.Duration) (string, error) {
	currentURL := ""
	if err := session.run(timeout, chromedp.Location(&currentURL)); err != nil {
		return "", err
	}
	return currentURL, nil
}

func browserRepairSessionByGoto(session *browserSession, targetURL string, timeout time.Duration) error {
	title, location := "", ""
	actions, err := browserGotoActions(targetURL, map[string]any{"wait": "load"}, &title, &location)
	if err != nil {
		return err
	}
	if err := session.run(timeout, actions...); err != nil {
		return err
	}
	session.rememberPageState(location, title)
	return nil
}

func browserContextRepairRequired(action browserActionKind) bool {
	switch action {
	case browserActionInfo, browserActionContent, browserActionEvaluate, browserActionScreenshot:
		return true
	default:
		return false
	}
}

func browserAutoRecoveryAllowed(action browserActionKind) bool {
	switch action {
	case browserActionInfo, browserActionContent, browserActionEvaluate, browserActionScreenshot, browserActionGoto:
		return true
	default:
		return false
	}
}

func isBrowserSessionInvalidError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	patterns := []string{
		browserSessionInvalidErrorCode,
		"context canceled",
		"context cancelled",
		"target closed",
		"session closed",
		"websocket: close",
		"websocket is closed",
		"connection reset by peer",
		"unexpected eof",
		"eof",
	}
	for _, pattern := range patterns {
		if strings.Contains(message, pattern) {
			return true
		}
	}
	return false
}

func wrapBrowserSessionInvalid(err error) error {
	if err == nil {
		err = errors.New("unknown browser session error")
	}
	payload := map[string]any{
		"code":    browserSessionInvalidErrorCode,
		"reason":  browserSessionInvalidReason(err),
		"message": strings.TrimSpace(err.Error()),
	}
	encoded, encodeErr := json.Marshal(payload)
	if encodeErr != nil {
		return fmt.Errorf("%s: %w", browserSessionInvalidErrorCode, err)
	}
	return errors.New(string(encoded))
}

// IsBrowserSessionInvalidToolError reports whether a tool error indicates browser session invalidation.
func IsBrowserSessionInvalidToolError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(
		strings.ToLower(strings.TrimSpace(err.Error())),
		browserSessionInvalidErrorCode,
	)
}

func browserSessionInvalidReason(err error) string {
	if err == nil {
		return "unknown"
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(message, "context canceled"), strings.Contains(message, "context cancelled"):
		return "context_canceled"
	case strings.Contains(message, "target closed"), strings.Contains(message, "session closed"):
		return "target_closed"
	case strings.Contains(message, "websocket"):
		return "websocket_closed"
	default:
		return "session_invalid"
	}
}

func browserResultWithRecovery(base map[string]any, meta browserRecoveryMeta) map[string]any {
	base["session_recovered"] = meta.Recovered
	if meta.Recovered {
		base["recovery_reason"] = meta.Reason
	}
	return base
}
