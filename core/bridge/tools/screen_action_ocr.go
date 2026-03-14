package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"ghost-os/bridge/tools/internal/tooljson"
	"ghost-os/bridge/tools/internal/toolparams"
)

var defaultScreenOCRLanguages = []string{"en", "zh"}

const (
	defaultScreenOCRConfidence = 0.75
	defaultScreenOCRCacheTTL   = time.Second
)

type screenOCRCache struct {
	mu    sync.Mutex
	entry *screenOCRCacheEntry
}

type screenOCRCacheEntry struct {
	key        screenOCRCacheKey
	payload    screenOCRPayload
	capturedAt time.Time
}

type screenOCRCacheKey struct {
	hasDisplayID  bool
	displayID     int
	hasRegion     bool
	region        screenRegion
	languages     []string
	minConfidence float64
}

func (t *ScreenActionTool) executeOCRScan(ctx context.Context, params map[string]any, traceID string) (string, error) {
	cacheKey, err := buildOCRCacheKey(params)
	if err != nil {
		return "", err
	}
	payload, err := t.execution.Call(ctx, "SCREEN_OCR", cloneParams(params), traceID)
	if err != nil {
		return "", fmt.Errorf("execution SCREEN_OCR failed: %w", err)
	}
	ocrPayload, err := tooljson.DecodePayload[screenOCRPayload](payload)
	if err != nil {
		return "", err
	}
	t.ensureOCRCache().store(cacheKey, ocrPayload)
	return tooljson.Encode(payload)
}

func (c *screenOCRCache) load(key screenOCRCacheKey, ttl time.Duration, reuse bool) (screenOCRPayload, bool) {
	if !reuse || ttl <= 0 {
		return screenOCRPayload{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entry == nil || !sameOCRCacheKey(c.entry.key, key) {
		return screenOCRPayload{}, false
	}
	if time.Since(c.entry.capturedAt) > ttl {
		return screenOCRPayload{}, false
	}
	return c.entry.payload, true
}

func (c *screenOCRCache) store(key screenOCRCacheKey, payload screenOCRPayload) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entry = &screenOCRCacheEntry{
		key:        key,
		payload:    payload,
		capturedAt: time.Now(),
	}
}

func buildOCRCacheKey(params map[string]any) (screenOCRCacheKey, error) {
	var key screenOCRCacheKey
	if len(params) == 0 {
		key.languages = defaultOCRLanguages()
		key.minConfidence = defaultOCRConfidence()
		return key, nil
	}

	if displayID, ok := toolparams.OptionalInt(params, "display_id"); ok {
		key.hasDisplayID = true
		key.displayID = displayID
	}
	if region, ok, err := parseOptionalRegion(params); err != nil {
		return key, err
	} else if ok {
		key.hasRegion = true
		key.region = region
	}

	languages, err := parseLanguages(params)
	if err != nil {
		return key, err
	}
	key.languages = languages

	if confidence, ok := toolparams.OptionalFloat(params, "min_confidence"); ok && confidence > 0 {
		key.minConfidence = confidence
		return key, nil
	}
	key.minConfidence = defaultOCRConfidence()
	return key, nil
}

func sameOCRCacheKey(left screenOCRCacheKey, right screenOCRCacheKey) bool {
	if left.hasDisplayID != right.hasDisplayID || left.displayID != right.displayID {
		return false
	}
	if left.hasRegion != right.hasRegion {
		return false
	}
	if left.hasRegion && left.region != right.region {
		return false
	}
	if left.minConfidence != right.minConfidence || len(left.languages) != len(right.languages) {
		return false
	}
	for index := range left.languages {
		if left.languages[index] != right.languages[index] {
			return false
		}
	}
	return true
}

func defaultOCRLanguages() []string {
	languages := make([]string, len(defaultScreenOCRLanguages))
	copy(languages, defaultScreenOCRLanguages)
	return languages
}

func defaultOCRConfidence() float64 {
	return defaultScreenOCRConfidence
}

func parseLanguages(params map[string]any) ([]string, error) {
	raw, ok := params["languages"]
	if !ok {
		return defaultOCRLanguages(), nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("languages must be an array")
	}

	normalized := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("languages must contain only strings")
		}
		text = strings.TrimSpace(strings.ToLower(text))
		if text != "" {
			normalized = append(normalized, text)
		}
	}
	if len(normalized) == 0 {
		return defaultOCRLanguages(), nil
	}

	sort.Strings(normalized)
	return compactStrings(normalized), nil
}

func compactStrings(items []string) []string {
	if len(items) < 2 {
		return items
	}
	out := items[:1]
	for _, item := range items[1:] {
		if item != out[len(out)-1] {
			out = append(out, item)
		}
	}
	return out
}

func parseCacheTTL(params map[string]any) time.Duration {
	return toolparams.DurationMillis(params, "cache_ttl_ms", defaultScreenOCRCacheTTL)
}

func cloneParams(params map[string]any) map[string]any {
	if len(params) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(params))
	for key, value := range params {
		cloned[key] = value
	}
	return cloned
}
