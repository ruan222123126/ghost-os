package orchestration

import (
	"errors"
	"sort"
	"strings"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/tools"
)

const (
	systemPromptActionGet    = "PROMPT_SYSTEM_GET"
	systemPromptActionUpdate = "PROMPT_SYSTEM_UPDATE"
)

type systemPromptResponse struct {
	CorePrompt     string                                 `json:"core_prompt"`
	RenderedPrompt string                                 `json:"rendered_prompt"`
	PromptLibrary  []bridgeconfig.SystemPromptLibraryItem `json:"prompt_library"`
}

type systemPromptPreviewCatalog struct {
	names []string
}

func (c systemPromptPreviewCatalog) Get(name string) tools.Tool {
	_ = name
	return nil
}

func (c systemPromptPreviewCatalog) ToolDefs() []llm.ToolDef {
	if len(c.names) == 0 {
		return nil
	}
	defs := make([]llm.ToolDef, 0, len(c.names))
	for _, name := range c.names {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		defs = append(defs, llm.ToolDef{Name: trimmed})
	}
	return defs
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

	rendered, err := s.renderSystemPromptPreview(cfg)
	if err != nil {
		logAction(traceID, systemPromptActionGet, "error", err)
		return ServiceResult{}, err
	}
	logAction(traceID, systemPromptActionGet, "success", nil)
	return serviceResultSuccess(systemPromptResponseFrom(files, rendered)), nil
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

	rendered, err := s.renderSystemPromptPreview(cfg)
	if err != nil {
		logAction(traceID, systemPromptActionUpdate, "error", err)
		return ServiceResult{}, err
	}
	logAction(traceID, systemPromptActionUpdate, "success", nil)
	return serviceResultSuccess(systemPromptResponseFrom(files, rendered)), nil
}

func (s *bridgeService) renderSystemPromptPreview(cfg bridgeconfig.Config) (string, error) {
	catalog := systemPromptPreviewCatalogFromConfig(cfg)
	if cfg.GraphQL.ToolRuntimeEnabled {
		catalog = systemPromptPreviewCatalog{
			names: systemPromptPreviewToolNames(cfg),
		}
		return bridgeruntime.BuildSystemPromptForCatalog(cfg, tools.NewStructuredToolHiddenCatalog(catalog))
	}
	return bridgeruntime.BuildSystemPromptForCatalog(cfg, catalog)
}

func systemPromptResponseFrom(files bridgeconfig.SystemPromptFiles, rendered string) systemPromptResponse {
	return systemPromptResponse{
		CorePrompt:     files.CorePrompt,
		RenderedPrompt: rendered,
		PromptLibrary:  files.PromptLibrary,
	}
}

func systemPromptPreviewCatalogFromConfig(cfg bridgeconfig.Config) systemPromptPreviewCatalog {
	return systemPromptPreviewCatalog{names: systemPromptPreviewToolNames(cfg)}
}

func systemPromptPreviewToolNames(cfg bridgeconfig.Config) []string {
	available := make([]string, 0, len(bridgeconfig.ToolBasePrompts()))
	for name := range bridgeconfig.ToolBasePrompts() {
		available = append(available, name)
	}
	sort.Strings(available)
	return bridgeruntime.NewToolSelectionPolicy(cfg).ResidentScope(available)
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
