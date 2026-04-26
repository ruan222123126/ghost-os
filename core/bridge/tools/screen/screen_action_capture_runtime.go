package screen

import (
	"context"
	"fmt"
	"sync"
	"time"

	"ghost-os/bridge/tools/internal/toolparams"
)

type screenCaptureCache struct {
	mu    sync.Mutex
	entry *screenCaptureCacheEntry
}

type screenCaptureCacheEntry struct {
	key        screenCaptureCacheKey
	payload    screenCapturePayload
	capturedAt time.Time
}

type screenCaptureCacheKey struct {
	hasDisplayID bool
	displayID    int
	hasRegion    bool
	region       screenRegion
}

func (t *ScreenActionTool) captureScreen(
	ctx context.Context,
	params map[string]any,
	traceID string,
) (screenCapturePayload, error) {
	return t.captureScreenInternal(ctx, params, traceID, true)
}

func (t *ScreenActionTool) captureScreenFresh(
	ctx context.Context,
	params map[string]any,
	traceID string,
) (screenCapturePayload, error) {
	return t.captureScreenInternal(ctx, params, traceID, false)
}

func (t *ScreenActionTool) captureScreenInternal(
	ctx context.Context,
	params map[string]any,
	traceID string,
	allowCache bool,
) (screenCapturePayload, error) {
	callParams, err := buildScreenCaptureParams(params)
	if err != nil {
		return screenCapturePayload{}, err
	}
	if !allowCache {
		return t.captureScreenOnce(ctx, callParams, traceID)
	}

	key, err := buildScreenCaptureCacheKey(callParams)
	if err != nil {
		return screenCapturePayload{}, err
	}
	ttl := parseCacheTTL(params)
	reuseCache := parseReuseCache(params)
	if cached, ok := t.ensureCaptureCache().load(key, ttl, reuseCache); ok {
		return cached, nil
	}

	captured, err := t.captureScreenOnce(ctx, callParams, traceID)
	if err != nil {
		return screenCapturePayload{}, err
	}
	if reuseCache && ttl > 0 {
		t.ensureCaptureCache().store(key, captured)
	}
	return captured, nil
}

func (t *ScreenActionTool) captureScreenOnce(
	ctx context.Context,
	callParams map[string]any,
	traceID string,
) (screenCapturePayload, error) {
	payload, err := t.execution.Call(ctx, "SCREEN_CAPTURE", callParams, traceID)
	if err != nil {
		return screenCapturePayload{}, fmt.Errorf("execution SCREEN_CAPTURE failed: %w", err)
	}
	return decodeScreenCapturePayload(payload)
}

func buildScreenCaptureCacheKey(callParams map[string]any) (screenCaptureCacheKey, error) {
	var key screenCaptureCacheKey
	if displayID, ok := toolparams.OptionalInt(callParams, "display_id"); ok {
		key.hasDisplayID = true
		key.displayID = displayID
	}
	rawRegion, ok := callParams["region"]
	if !ok {
		return key, nil
	}

	region, ok := rawRegion.(screenRegion)
	if !ok {
		return key, fmt.Errorf("invalid screen capture cache key: region must be screenRegion")
	}
	key.hasRegion = true
	key.region = region
	return key, nil
}

func parseReuseCache(params map[string]any) bool {
	return toolparams.OptionalBool(params, "reuse_cache", true)
}

func (c *screenCaptureCache) load(
	key screenCaptureCacheKey,
	ttl time.Duration,
	reuse bool,
) (screenCapturePayload, bool) {
	if !reuse || ttl <= 0 {
		return screenCapturePayload{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entry == nil || !sameScreenCaptureCacheKey(c.entry.key, key) {
		return screenCapturePayload{}, false
	}
	if time.Since(c.entry.capturedAt) > ttl {
		return screenCapturePayload{}, false
	}
	return c.entry.payload, true
}

func (c *screenCaptureCache) store(key screenCaptureCacheKey, payload screenCapturePayload) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entry = &screenCaptureCacheEntry{
		key:        key,
		payload:    payload,
		capturedAt: time.Now(),
	}
}

func sameScreenCaptureCacheKey(left screenCaptureCacheKey, right screenCaptureCacheKey) bool {
	if left.hasDisplayID != right.hasDisplayID || left.displayID != right.displayID {
		return false
	}
	if left.hasRegion != right.hasRegion {
		return false
	}
	if left.hasRegion && left.region != right.region {
		return false
	}
	return true
}
