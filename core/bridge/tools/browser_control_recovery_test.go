package tools

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

const (
	testBrowserPreflightTimeout     = 200 * time.Millisecond
	testBrowserFastPreflightTimeout = 50 * time.Millisecond
)

func TestBrowserRunWithRecoveryRetriesOnceForGoto(t *testing.T) {
	tool := NewBrowserControlToolWithPool(nil, NewBrowserSessionPool()).(*BrowserControlTool)
	scope := browserSessionGlobalScope
	mockBrowserWSEndpointFetcher(
		t,
		func(base string, _ time.Duration) (string, error) {
			return strings.Replace(base, "http://", "ws://", 1) + "/devtools/browser/discovered", nil
		},
	)
	tool.storeSession(scope, &browserSession{
		id:         "session-1",
		scopeKey:   scope,
		wsEndpoint: "ws://127.0.0.1:9222/devtools/browser/test-1",
	})
	defer closeBrowserSessionInScope(t, tool, scope, "session-1")

	runCount := 0
	session, err := tool.sessionByID(scope, "session-1")
	if err != nil {
		t.Fatalf("sessionByID returned error: %v", err)
	}
	meta, err := tool.runWithRecovery(
		context.Background(),
		browserActionGoto,
		session,
		testBrowserPreflightTimeout,
		func(_ *browserSession) error {
			runCount++
			if runCount == 1 {
				return errors.New("context canceled")
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("runWithRecovery returned error: %v", err)
	}
	if runCount != 2 {
		t.Fatalf("expected one retry, got run count %d", runCount)
	}
	if !meta.Recovered || meta.Reason != "context_canceled" {
		t.Fatalf("unexpected recovery meta: %+v", meta)
	}
}

func TestBrowserRunWithRecoveryRefreshesWSEndpoint(t *testing.T) {
	tool := NewBrowserControlToolWithPool(nil, NewBrowserSessionPool()).(*BrowserControlTool)
	scope := browserSessionGlobalScope
	refreshedEndpoint := "ws://127.0.0.1:9222/devtools/browser/fresh"
	mockBrowserWSEndpointFetcher(
		t,
		func(base string, _ time.Duration) (string, error) {
			return strings.Replace(base, "http://", "ws://", 1) + "/devtools/browser/fresh", nil
		},
	)
	tool.storeSession(scope, &browserSession{
		id:         "session-refresh",
		scopeKey:   scope,
		wsEndpoint: "ws://127.0.0.1:9222/devtools/browser/stale",
	})
	defer closeBrowserSessionInScope(t, tool, scope, "session-refresh")

	runCount := 0
	observedEndpoint := ""
	session, err := tool.sessionByID(scope, "session-refresh")
	if err != nil {
		t.Fatalf("sessionByID returned error: %v", err)
	}
	_, err = tool.runWithRecovery(
		context.Background(),
		browserActionGoto,
		session,
		testBrowserPreflightTimeout,
		func(active *browserSession) error {
			runCount++
			if runCount == 1 {
				return errors.New("context canceled")
			}
			observedEndpoint = active.wsEndpoint
			return nil
		},
	)
	if err != nil {
		t.Fatalf("runWithRecovery returned error: %v", err)
	}
	if observedEndpoint != refreshedEndpoint {
		t.Fatalf("expected refreshed ws endpoint %q, got %q", refreshedEndpoint, observedEndpoint)
	}
}

func TestBrowserRunWithRecoveryDoesNotRetryClick(t *testing.T) {
	tool := NewBrowserControlToolWithPool(nil, NewBrowserSessionPool()).(*BrowserControlTool)
	scope := browserSessionGlobalScope
	mockBrowserWSEndpointFetcher(
		t,
		func(base string, _ time.Duration) (string, error) {
			return strings.Replace(base, "http://", "ws://", 1) + "/devtools/browser/discovered", nil
		},
	)
	tool.storeSession(scope, &browserSession{
		id:         "session-2",
		scopeKey:   scope,
		wsEndpoint: "ws://127.0.0.1:9222/devtools/browser/test-2",
	})
	defer closeBrowserSessionInScope(t, tool, scope, "session-2")

	runCount := 0
	session, err := tool.sessionByID(scope, "session-2")
	if err != nil {
		t.Fatalf("sessionByID returned error: %v", err)
	}
	_, err = tool.runWithRecovery(
		context.Background(),
		browserActionClick,
		session,
		testBrowserPreflightTimeout,
		func(_ *browserSession) error {
			runCount++
			return errors.New("context canceled")
		},
	)
	if err == nil {
		t.Fatal("expected browser session invalid error")
	}
	if runCount != 1 {
		t.Fatalf("expected no retry, got run count %d", runCount)
	}
	if !strings.Contains(err.Error(), `"code":"browser_session_invalid"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBrowserRunWithRecoveryStopsWhenPreflightFails(t *testing.T) {
	tool := NewBrowserControlToolWithPool(nil, NewBrowserSessionPool()).(*BrowserControlTool)
	scope := browserSessionGlobalScope
	mockBrowserWSEndpointFetcher(
		t,
		func(base string, _ time.Duration) (string, error) {
			return "", errors.New("fetch " + browserVersionURL(base) + " failed: dial tcp 127.0.0.1:9223: connect: connection refused")
		},
	)
	tool.storeSession(scope, &browserSession{
		id:         "session-preflight-fail",
		scopeKey:   scope,
		wsEndpoint: "ws://127.0.0.1:9223/devtools/browser/missing",
	})
	defer closeBrowserSessionInScope(t, tool, scope, "session-preflight-fail")

	runCount := 0
	session, err := tool.sessionByID(scope, "session-preflight-fail")
	if err != nil {
		t.Fatalf("sessionByID returned error: %v", err)
	}
	_, err = tool.runWithRecovery(
		context.Background(),
		browserActionInfo,
		session,
		testBrowserFastPreflightTimeout,
		func(_ *browserSession) error {
			runCount++
			return nil
		},
	)
	if err == nil {
		t.Fatal("expected preflight error")
	}
	if runCount != 0 {
		t.Fatalf("expected run to be skipped after preflight failure, got run count %d", runCount)
	}
	if !strings.Contains(err.Error(), "browser cdp preflight failed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "/json/version") {
		t.Fatalf("expected /json/version probe diagnostics, got: %v", err)
	}
}

func TestBrowserGotoActionsIncludeReadySelectorWait(t *testing.T) {
	var title string
	var location string
	baseActions, err := browserGotoActions("https://example.com", map[string]any{}, &title, &location)
	if err != nil {
		t.Fatalf("browserGotoActions returned error: %v", err)
	}
	selectorActions, err := browserGotoActions(
		"https://example.com",
		map[string]any{"ready_selector": "#app"},
		&title,
		&location,
	)
	if err != nil {
		t.Fatalf("browserGotoActions with ready selector returned error: %v", err)
	}
	if len(selectorActions) <= len(baseActions) {
		t.Fatalf("expected extra ready selector wait action, got base=%d withSelector=%d", len(baseActions), len(selectorActions))
	}
}

func TestBrowserShouldRepairSessionByGoto(t *testing.T) {
	cases := []struct {
		name       string
		currentURL string
		err        error
		want       bool
	}{
		{
			name:       "blank url without error",
			currentURL: "about:blank",
			err:        nil,
			want:       true,
		},
		{
			name:       "normal url without error",
			currentURL: "https://example.com",
			err:        nil,
			want:       false,
		},
		{
			name:       "session invalid error",
			currentURL: "",
			err:        errors.New("context canceled"),
			want:       true,
		},
		{
			name:       "non session error",
			currentURL: "",
			err:        errors.New("invalid selector"),
			want:       false,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := browserShouldRepairSessionByGoto(tc.currentURL, tc.err)
			if got != tc.want {
				t.Fatalf("unexpected repair decision: got %v want %v", got, tc.want)
			}
		})
	}
}
