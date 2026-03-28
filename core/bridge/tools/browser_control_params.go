package tools

import (
	"time"

	"ghost-os/bridge/tools/internal/toolparams"
)

const (
	defaultBrowserLaunchWaitTimeout  = 8 * time.Second
	defaultBrowserEndpointTimeout    = 5 * time.Second
	defaultBrowserFetchTimeout       = 3 * time.Second
	defaultBrowserEndpointPoll       = 200 * time.Millisecond
	defaultBrowserDebugPort          = 9222
	maxBrowserDebugPort              = 65535
	browserAutoLaunchProfileDir      = "/tmp/ghost-browser-control-profile"
	browserAutoLaunchLogPath         = "/tmp/ghost-browser-control.log"
	autoBrowserLaunchCommandTemplate = `set -euo pipefail
BROWSER_BIN=""
for candidate in %s; do
	if command -v "$candidate" >/dev/null 2>&1; then
		BROWSER_BIN="$candidate"
		break
	fi
done
if [ -z "$BROWSER_BIN" ]; then
	echo "no Chrome-compatible browser binary found in PATH (tried: %s)" >&2
	exit 127
fi
"$BROWSER_BIN" --headless --disable-gpu --remote-debugging-address=127.0.0.1 --remote-debugging-port=%d --no-first-run --no-default-browser-check --disable-dev-shm-usage --no-sandbox --user-data-dir=%s about:blank >%s 2>&1 &
echo "browser_binary=$BROWSER_BIN"`
)

var browserAutoLaunchCandidates = []string{
	"google-chrome",
	"google-chrome-stable",
	"chromium",
	"chromium-browser",
	"chrome",
}

var missingBinaryMarkers = []string{
	"command not found",
	"not recognized as an internal or external command",
	"is not recognized as an internal or external command",
	"no such file or directory",
}

func browserParseTimeout(params map[string]any) time.Duration {
	return toolparams.DurationMillis(params, "timeout_ms", 0)
}

func browserParseWaitDuration(params map[string]any, field string) time.Duration {
	return toolparams.DurationMillis(params, field, 0)
}

func browserParseWaitTimeout(params map[string]any) time.Duration {
	return toolparams.DurationMillis(params, "wait_timeout_ms", defaultBrowserLaunchWaitTimeout)
}
