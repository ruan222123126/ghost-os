package orchestration

import (
	"errors"
	"fmt"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/orchestration/internal/contracts/toolschema"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/tools"
)

const (
	systemPromptActionGet    = "PROMPT_SYSTEM_GET"
	systemPromptActionUpdate = "PROMPT_SYSTEM_UPDATE"
)

type systemPromptResponse struct {
	CorePrompt      string                                 `json:"core_prompt"`
	RenderedPrompt  string                                 `json:"rendered_prompt"`
	PromptLibrary   []bridgeconfig.SystemPromptLibraryItem `json:"prompt_library"`
	ToolDefinitions []systemPromptToolDefinition           `json:"tool_definitions"`
}

type systemPromptToolDefinition = toolschema.Definition

type systemPromptPreview struct {
	renderedPrompt  string
	toolDefinitions []systemPromptToolDefinition
}

func (s *bridgeService) executeSystemPromptGetAction(traceID string) (ServiceResult, error) {
	cfg, err := s.configStore.Config()
	if err != nil {
		logAction(traceID, systemPromptActionGet, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInternal, err)
	}

	files, err := bridgeconfig.LoadSystemPromptFiles(cfg.PromptsDir)
	if err != nil {
		logAction(traceID, systemPromptActionGet, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInternal, err)
	}

	preview, err := s.loadSystemPromptPreview(cfg)
	if err != nil {
		logAction(traceID, systemPromptActionGet, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInternal, err)
	}
	logAction(traceID, systemPromptActionGet, "success", nil)
	return serviceResultSuccess(systemPromptResponseFrom(files, preview)), nil
}

func (s *bridgeService) executeSystemPromptUpdateAction(
	req bridgeconfig.SystemPromptUpdateRequest,
	traceID string,
) (ServiceResult, error) {
	if !reqHasSystemPromptUpdate(req) {
		err := bridgeconfig.ErrSystemPromptUpdateEmpty
		logAction(traceID, systemPromptActionUpdate, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInvalidInput, err)
	}

	cfg, err := s.configStore.Config()
	if err != nil {
		logAction(traceID, systemPromptActionUpdate, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInternal, err)
	}

	files, err := bridgeconfig.UpdateSystemPromptFiles(cfg.PromptsDir, req)
	if err != nil {
		logAction(traceID, systemPromptActionUpdate, "error", err)
		return ServiceResult{}, mapSystemPromptError(err)
	}

	preview, err := s.loadSystemPromptPreview(cfg)
	if err != nil {
		logAction(traceID, systemPromptActionUpdate, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInternal, err)
	}
	logAction(traceID, systemPromptActionUpdate, "success", nil)
	return serviceResultSuccess(systemPromptResponseFrom(files, preview)), nil
}

func (s *bridgeService) loadSystemPromptPreview(cfg bridgeconfig.Config) (systemPromptPreview, error) {
	catalog, cleanup, err := s.loadSystemPromptPreviewCatalog(cfg)
	if err != nil {
		return systemPromptPreview{}, err
	}
	defer cleanup()

	rendered, err := bridgeruntime.BuildSystemPromptForCatalog(cfg, catalog)
	if err != nil {
		return systemPromptPreview{}, fmt.Errorf("build system prompt preview: %w", err)
	}
	return systemPromptPreview{
		renderedPrompt:  rendered,
		toolDefinitions: systemPromptToolDefinitionsFrom(catalog.ToolDefs()),
	}, nil
}

func (s *bridgeService) loadSystemPromptPreviewCatalog(
	cfg bridgeconfig.Config,
) (tools.ToolCatalog, func(), error) {
	if s == nil || s.runtimeFactory == nil {
		return nil, func() {}, errors.New("runtime factory unavailable for system prompt preview")
	}

	deps, err := s.runtimeFactory.Build(s.configStore)
	if err != nil {
		return nil, func() {}, fmt.Errorf("build runtime dependencies for system prompt preview: %w", err)
	}
	if deps.registry == nil {
		deps.Close()
		return nil, func() {}, errors.New("tool registry unavailable for system prompt preview")
	}
	baseCatalog := tools.NewPromptOverrideCatalog(deps.registry, deps.cfg.ToolSelector.PromptOverrides)
	return bridgeruntime.NewToolSelectionPolicy(cfg).ResidentCatalog(baseCatalog), deps.Close, nil
}

func systemPromptResponseFrom(
	files bridgeconfig.SystemPromptFiles,
	preview systemPromptPreview,
) systemPromptResponse {
	return systemPromptResponse{
		CorePrompt:      files.CorePrompt,
		RenderedPrompt:  preview.renderedPrompt,
		PromptLibrary:   files.PromptLibrary,
		ToolDefinitions: preview.toolDefinitions,
	}
}

func systemPromptToolDefinitionsFrom(defs []llm.ToolDef) []systemPromptToolDefinition {
	return toolschema.DefinitionsFrom(defs)
}

func reqHasSystemPromptUpdate(req bridgeconfig.SystemPromptUpdateRequest) bool {
	return req.CorePrompt != nil || req.PromptLibrary != nil
}

func mapSystemPromptError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, bridgeconfig.ErrSystemPromptUpdateEmpty) {
		return wrapServiceError(ServiceErrorInvalidInput, err)
	}
	if errors.Is(err, bridgeconfig.ErrSystemPromptUpdateConflict) {
		return wrapServiceError(ServiceErrorInvalidInput, err)
	}
	if errors.Is(err, bridgeconfig.ErrSystemPromptLibraryInvalid) {
		return wrapServiceError(ServiceErrorInvalidInput, err)
	}
	return wrapServiceError(ServiceErrorInternal, err)
}
