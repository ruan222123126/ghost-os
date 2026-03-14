package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ghost-os/bridge/tools/internal/tooljson"
	"ghost-os/bridge/tools/internal/toolparams"
)

func (t *BrowserControlTool) executeConnect(params map[string]any) (string, error) {
	endpoint, err := toolparams.RequiredString(params, "endpoint")
	if err != nil {
		return "", err
	}
	sessionID := toolparams.OptionalString(params, "session_id", "")
	if sessionID == "" {
		sessionID = newBrowserSessionID()
	}

	wsEndpoint, err := resolveWSEndpoint(endpoint, browserParseTimeout(params))
	if err != nil {
		return "", err
	}
	session, err := newRemoteBrowserSession(sessionID, wsEndpoint)
	if err != nil {
		return "", err
	}
	t.storeSession(session)
	return tooljson.Encode(map[string]any{
		"action":      "connect",
		"session_id":  sessionID,
		"ws_endpoint": wsEndpoint,
		"connected":   true,
	})
}

func (t *BrowserControlTool) executeLaunch(ctx context.Context, params map[string]any, traceID string) (string, error) {
	if t.execution == nil {
		return "", fmt.Errorf("execution client is not configured")
	}
	command, err := toolparams.RequiredString(params, "command")
	if err != nil {
		return "", err
	}

	if _, err := t.execution.Call(ctx, "BASH_EXEC", map[string]any{"command": command}, traceID); err != nil {
		return "", fmt.Errorf("launch command failed: %w", err)
	}
	endpoint, err := launchEndpoint(params)
	if err != nil {
		return "", err
	}

	wsEndpoint, err := waitForWSEndpoint(endpoint, browserParseWaitTimeout(params))
	if err != nil {
		return "", err
	}
	sessionID := toolparams.OptionalString(params, "session_id", "")
	if sessionID == "" {
		sessionID = newBrowserSessionID()
	}
	if err := t.replaceSession(sessionID, wsEndpoint); err != nil {
		return "", err
	}
	return tooljson.Encode(map[string]any{
		"action":      "launch",
		"session_id":  sessionID,
		"ws_endpoint": wsEndpoint,
		"connected":   true,
	})
}

func launchEndpoint(params map[string]any) (string, error) {
	endpoint := toolparams.OptionalString(params, "endpoint", "")
	if endpoint != "" {
		return endpoint, nil
	}
	port, ok := toolparams.OptionalInt(params, "debug_port")
	if !ok {
		return "", fmt.Errorf("debug_port or endpoint is required for launch")
	}
	return fmt.Sprintf("http://127.0.0.1:%d", port), nil
}

func resolveWSEndpoint(endpoint string, timeout time.Duration) (string, error) {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		return "", fmt.Errorf("endpoint is required")
	}
	if strings.HasPrefix(trimmed, "ws://") || strings.HasPrefix(trimmed, "wss://") {
		return trimmed, nil
	}
	if !strings.Contains(trimmed, "://") {
		trimmed = "http://" + trimmed
	}
	return fetchWebSocketURL(trimmed, timeout)
}

func waitForWSEndpoint(endpoint string, timeout time.Duration) (string, error) {
	if timeout <= 0 {
		timeout = defaultBrowserEndpointTimeout
	}
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		wsEndpoint, err := resolveWSEndpoint(endpoint, timeout)
		if err == nil {
			return wsEndpoint, nil
		}
		lastErr = err
		time.Sleep(defaultBrowserEndpointPoll)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("timed out waiting for endpoint")
	}
	return "", lastErr
}

func fetchWebSocketURL(base string, timeout time.Duration) (string, error) {
	return fetchWebSocketURLWithClient(base, &http.Client{Timeout: httpClientTimeout(timeout)})
}

func fetchWebSocketURLWithClient(base string, client *http.Client) (string, error) {
	if client == nil {
		return "", fmt.Errorf("http client is required")
	}
	url := strings.TrimRight(base, "/")
	if !strings.HasSuffix(url, "/json/version") {
		url += "/json/version"
	}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("fetch %s failed: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("fetch %s failed: status %d", url, resp.StatusCode)
	}

	var payload struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode %s failed: %w", url, err)
	}
	if strings.TrimSpace(payload.WebSocketDebuggerURL) == "" {
		return "", fmt.Errorf("missing webSocketDebuggerUrl in %s", url)
	}
	return payload.WebSocketDebuggerURL, nil
}

func httpClientTimeout(timeout time.Duration) time.Duration {
	if timeout > 0 {
		return timeout
	}
	return defaultBrowserFetchTimeout
}
