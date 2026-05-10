package orchestration

import (
	"errors"
	"fmt"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	appprompts "ghost-os/bridge/orchestration/internal/app/prompts"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/tools"
)

const (
	systemPromptActionGet    = appprompts.SystemPromptActionGet
	systemPromptActionUpdate = appprompts.SystemPromptActionUpdate
)

type systemPromptResponse = appprompts.SystemResponse

type systemPromptToolDefinition = appprompts.ToolDefinition

type systemPromptPreview = appprompts.Preview

func (s *bridgeService) promptService() appprompts.Service {
	return appprompts.Service{
		Store:         s.configStore,
		PreviewLoader: systemPromptPreviewLoader{service: s},
		Logger:        serviceActionLogger{},
	}
}

type serviceActionLogger struct{}

func (serviceActionLogger) Log(traceID string, action string, status string, err error) {
	logAction(traceID, action, status, err)
}

type systemPromptPreviewLoader struct {
	service *bridgeService
}

func (l systemPromptPreviewLoader) Load(cfg bridgeconfig.Config) (appprompts.Preview, error) {
	if l.service == nil {
		return appprompts.Preview{}, errors.New("service is not configured")
	}
	return l.service.loadSystemPromptPreview(cfg)
}

func (s *bridgeService) executeSystemPromptGetAction(traceID string) (ServiceResult, error) {
	return s.promptService().GetSystemPrompt(traceID)
}

func (s *bridgeService) executeSystemPromptUpdateAction(
	req bridgeconfig.SystemPromptUpdateRequest,
	traceID string,
) (ServiceResult, error) {
	return s.promptService().UpdateSystemPrompt(req, traceID)
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
		RenderedPrompt:  rendered,
		ToolDefinitions: systemPromptToolDefinitionsFrom(catalog.ToolDefs()),
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
		RenderedPrompt:  preview.RenderedPrompt,
		PromptLibrary:   files.PromptLibrary,
		ToolDefinitions: preview.ToolDefinitions,
	}
}

func systemPromptToolDefinitionsFrom(defs []llm.ToolDef) []systemPromptToolDefinition {
	return appprompts.ToolDefinitionsFrom(defs)
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
