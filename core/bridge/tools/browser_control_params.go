package tools

import (
	"time"

	"ghost-os/bridge/tools/internal/toolparams"
)

const (
	defaultBrowserLaunchWaitTimeout = 8 * time.Second
	defaultBrowserEndpointTimeout   = 5 * time.Second
	defaultBrowserFetchTimeout      = 3 * time.Second
	defaultBrowserEndpointPoll      = 200 * time.Millisecond
)

func browserParseTimeout(params map[string]any) time.Duration {
	return toolparams.DurationMillis(params, "timeout_ms", 0)
}

func browserParseWaitDuration(params map[string]any, field string) time.Duration {
	return toolparams.DurationMillis(params, field, 0)
}

func browserParseWaitTimeout(params map[string]any) time.Duration {
	return toolparams.DurationMillis(params, "wait_timeout_ms", defaultBrowserLaunchWaitTimeout)
}
