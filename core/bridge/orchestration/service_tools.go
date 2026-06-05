package orchestration

import (
	"context"
	"encoding/json"
	"fmt"

	"ghost-os/bridge/llm"
	apptools "ghost-os/bridge/orchestration/internal/app/tools"
	"ghost-os/bridge/tools"
)

const (
	busActionToolList   = apptools.ActionList
	busActionToolUpdate = apptools.ActionUpdate

	findIconTemplateUploadAction = apptools.FindIconTemplateUploadAction
	findIconPreviewAction        = apptools.FindIconPreviewAction
	findIconTemplateRoot         = apptools.FindIconTemplateRoot
	screenControlToolID          = apptools.ScreenControlToolID
)

func (s *bridgeService) executeToolListAction(traceID string) (any, int, error) {
	return s.toolService().List(traceID)
}

func (s *bridgeService) executeToolUpdateAction(
	params toolNameParams,
	req toolUpdateRequest,
	traceID string,
) (any, int, error) {
	return s.toolService().Update(params, req, traceID)
}

func (s *bridgeService) toolService() apptools.Service {
	return apptools.Service{
		Store:          s.configStore,
		SchemaProvider: toolSchemaProvider{service: s},
		Logger:         serviceActionLogger{},
	}
}

type toolSchemaProvider struct {
	service *bridgeService
}

func (p toolSchemaProvider) ListToolInputSchemas() (map[string]map[string]any, error) {
	return p.service.listToolInputSchemas()
}

func mapToolConfigError(err error) int {
	return apptools.MapConfigError(err)
}

func cloneOptionalInt(raw *int) *int {
	return apptools.CloneOptionalInt(raw)
}

func (s *bridgeService) listToolInputSchemas() (map[string]map[string]any, error) {
	if s == nil || s.runtimeFactory == nil {
		return nil, nil
	}
	deps, err := s.runtimeFactory.Build(s.configStore)
	if err != nil {
		return nil, err
	}
	defer deps.Close()
	if deps.registry == nil {
		return map[string]map[string]any{}, nil
	}
	return collectToolSchemas(deps.registry.ToolDefs())
}

func collectToolSchemas(defs []llm.ToolDef) (map[string]map[string]any, error) {
	return apptools.CollectSchemas(defs)
}

func decodeToolSchema(raw json.RawMessage, toolName string) (map[string]any, error) {
	return apptools.DecodeSchema(raw, toolName)
}

func schemaByToolName(schemasByName map[string]map[string]any, toolName string) (map[string]any, bool) {
	schema := apptools.SchemaByToolName(schemasByName, toolName)
	return schema, schema != nil
}

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

const mousePositionAction = apptools.MousePositionAction

func (s *bridgeService) executeMousePositionActionResult(
	ctx context.Context,
	_ mousePositionRequest,
	traceID string,
) (ServiceResult, error) {
	payload, err := s.runMousePosition(ctx, traceID)
	if err != nil {
		logAction(traceID, mousePositionAction, "error", err)
		return ServiceResult{}, err
	}
	logAction(traceID, mousePositionAction, "success", nil)
	return serviceResultSuccess(payload), nil
}

func (s *bridgeService) runMousePosition(
	ctx context.Context,
	traceID string,
) (mousePositionPayload, error) {
	tool, cleanup, err := s.screenControlTool()
	if err != nil {
		return mousePositionPayload{}, err
	}
	defer cleanup()
	return executeMousePositionToolCall(ctx, tool, traceID)
}

func executeMousePositionToolCall(
	ctx context.Context,
	tool tools.Tool,
	traceID string,
) (mousePositionPayload, error) {
	return apptools.ExecuteMousePositionToolCall(ctx, tool, traceID)
}

func buildMousePositionToolArgs() map[string]any {
	return apptools.BuildMousePositionToolArgs()
}

func decodeMousePositionPayload(raw string) (mousePositionPayload, error) {
	return apptools.DecodeMousePositionPayload(raw)
}

func parseOptionalMousePositionInt(raw any) (int, bool) {
	return apptools.ParseOptionalMousePositionInt(raw)
}

func parseOptionalMousePositionFloat(raw any) (float64, bool) {
	return apptools.ParseOptionalMousePositionFloat(raw)
}

func (s *bridgeService) executeFindIconTemplateUploadActionResult(
	req findIconTemplateUploadRequest,
	traceID string,
) (ServiceResult, error) {
	payload, err := executeFindIconTemplateUpload(req)
	if err != nil {
		logAction(traceID, findIconTemplateUploadAction, "error", err)
		return ServiceResult{}, err
	}
	logAction(traceID, findIconTemplateUploadAction, "success", nil)
	return serviceResultCreated(payload), nil
}

func executeFindIconTemplateUpload(req findIconTemplateUploadRequest) (findIconTemplateUploadPayload, error) {
	return apptools.ExecuteFindIconTemplateUpload(req)
}

func resolveFindIconTemplateRoot() (string, error) {
	return apptools.ResolveFindIconTemplateRoot()
}

func (s *bridgeService) executeFindIconPreviewActionResult(
	ctx context.Context,
	req findIconPreviewRequest,
	traceID string,
) (ServiceResult, error) {
	params, err := normalizeFindIconPreviewRequest(req)
	if err != nil {
		logAction(traceID, findIconPreviewAction, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInvalidInput, err)
	}
	payload, err := s.runFindIconPreview(ctx, params, traceID)
	if err != nil {
		logAction(traceID, findIconPreviewAction, "error", err)
		return ServiceResult{}, err
	}
	logAction(traceID, findIconPreviewAction, "success", nil)
	return serviceResultSuccess(payload), nil
}

func normalizeFindIconPreviewRequest(req findIconPreviewRequest) (findIconPreviewRequest, error) {
	return apptools.NormalizeFindIconPreviewRequest(req)
}

func (s *bridgeService) runFindIconPreview(
	ctx context.Context,
	req findIconPreviewRequest,
	traceID string,
) (findIconPreviewPayload, error) {
	tool, cleanup, err := s.screenControlTool()
	if err != nil {
		return findIconPreviewPayload{}, err
	}
	defer cleanup()
	return executeFindIconPreviewToolCall(ctx, tool, req, traceID)
}

func (s *bridgeService) screenControlTool() (tools.Tool, func(), error) {
	if s == nil || s.runtimeFactory == nil {
		return nil, func() {}, wrapServiceError(ServiceErrorUnavailable, fmt.Errorf("runtime factory is not configured"))
	}
	deps, err := s.runtimeFactory.Build(s.configStore)
	if err != nil {
		return nil, func() {}, wrapServiceError(ServiceErrorUnavailable, fmt.Errorf("build runtime dependencies: %w", err))
	}
	if deps.registry == nil {
		deps.Close()
		return nil, func() {}, wrapServiceError(ServiceErrorUnavailable, fmt.Errorf("tool registry is not configured"))
	}
	tool := deps.registry.Get(screenControlToolID)
	if tool == nil {
		deps.Close()
		return nil, func() {}, wrapServiceError(ServiceErrorUnavailable, fmt.Errorf("tool %q is not available", screenControlToolID))
	}
	return tool, deps.Close, nil
}

func executeFindIconPreviewToolCall(
	ctx context.Context,
	tool tools.Tool,
	req findIconPreviewRequest,
	traceID string,
) (findIconPreviewPayload, error) {
	return apptools.ExecuteFindIconPreviewToolCall(ctx, tool, req, traceID)
}
