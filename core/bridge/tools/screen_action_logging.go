package tools

import (
	"log"
	"strings"

	"ghost-os/bridge/tools/internal/toolparams"
)

func logClickText(traceID string, query string, params map[string]any) {
	log.Printf(
		"trace_id=%s tool=screen_action action=click_text direct=false x=%v y=%v query=%q",
		strings.TrimSpace(traceID),
		params["x"],
		params["y"],
		query,
	)
}

func logClickIcon(traceID string, direct bool, params map[string]any) {
	log.Printf(
		"trace_id=%s tool=screen_action action=click_icon direct=%t x=%v y=%v template_path=%q",
		strings.TrimSpace(traceID),
		direct,
		params["x"],
		params["y"],
		toolparams.OptionalString(params, "template_path", ""),
	)
}
