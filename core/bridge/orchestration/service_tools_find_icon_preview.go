package orchestration

import (
	"context"

	apptools "ghost-os/bridge/orchestration/internal/app/tools"
	"ghost-os/bridge/tools"
)

func buildFindIconPreviewToolArgs(req findIconPreviewRequest) map[string]any {
	return apptools.BuildFindIconPreviewToolArgs(req)
}

func executeFindIconPreviewHover(
	ctx context.Context,
	tool tools.Tool,
	payload findIconPreviewPayload,
	traceID string,
) error {
	return apptools.ExecuteFindIconPreviewHover(ctx, tool, payload, traceID)
}

func buildFindIconPreviewHoverToolArgs(x int, y int, displayID *int) map[string]any {
	return apptools.BuildFindIconPreviewHoverToolArgs(x, y, displayID)
}

func firstFindIconMatchCenter(matches []map[string]any) (int, int, error) {
	return apptools.FirstFindIconMatchCenter(matches)
}

func decodeFindIconPreviewPayload(raw string) (findIconPreviewPayload, error) {
	return apptools.DecodeFindIconPreviewPayload(raw)
}

func parseFindIconMatchList(raw any) ([]map[string]any, error) {
	return apptools.ParseFindIconMatchList(raw)
}

func parseOptionalFindIconInt(raw any) (int, bool) {
	return apptools.ParseOptionalFindIconInt(raw)
}

func parseOptionalFindIconRegion(raw any) (findIconPreviewRegion, bool, error) {
	return apptools.ParseOptionalFindIconRegion(raw)
}
