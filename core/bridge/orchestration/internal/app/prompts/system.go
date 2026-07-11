package prompts

import (
	"errors"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
	"ghost-os/bridge/orchestration/internal/contracts/toolschema"
)

const (
	SystemPromptActionGet    = "PROMPT_SYSTEM_GET"
	SystemPromptActionUpdate = "PROMPT_SYSTEM_UPDATE"
)

type ToolDefinition = toolschema.Definition

type SystemResponse struct {
	CorePrompt      string                                 `json:"core_prompt"`
	RenderedPrompt  string                                 `json:"rendered_prompt"`
	PromptLibrary   []bridgeconfig.SystemPromptLibraryItem `json:"prompt_library"`
	ToolDefinitions []ToolDefinition                       `json:"tool_definitions"`
}

type Preview struct {
	RenderedPrompt  string
	ToolDefinitions []ToolDefinition
}

type PreviewLoader interface {
	Load(cfg bridgeconfig.Config) (Preview, error)
}

func (s Service) GetSystemPrompt(traceID string) (bus.ServiceResult, error) {
	cfg, files, err := s.loadSystemFiles()
	if err != nil {
		s.log(traceID, SystemPromptActionGet, "error", err)
		return bus.ServiceResult{}, bus.WrapError(bus.ServiceErrorInternal, err)
	}
	preview, err := s.PreviewLoader.Load(cfg)
	if err != nil {
		s.log(traceID, SystemPromptActionGet, "error", err)
		return bus.ServiceResult{}, bus.WrapError(bus.ServiceErrorInternal, err)
	}
	s.log(traceID, SystemPromptActionGet, "success", nil)
	return bus.ResultSuccess(systemResponseFrom(files, preview)), nil
}

func (s Service) UpdateSystemPrompt(
	req bridgeconfig.SystemPromptUpdateRequest,
	traceID string,
) (bus.ServiceResult, error) {
	if !hasSystemPromptUpdate(req) {
		err := bridgeconfig.ErrSystemPromptUpdateEmpty
		s.log(traceID, SystemPromptActionUpdate, "error", err)
		return bus.ServiceResult{}, bus.WrapError(bus.ServiceErrorInvalidInput, err)
	}
	cfg, err := s.config()
	if err != nil {
		s.log(traceID, SystemPromptActionUpdate, "error", err)
		return bus.ServiceResult{}, bus.WrapError(bus.ServiceErrorInternal, err)
	}
	files, err := bridgeconfig.UpdateSystemPromptFiles(cfg.PromptsDir, req)
	if err != nil {
		s.log(traceID, SystemPromptActionUpdate, "error", err)
		return bus.ServiceResult{}, mapSystemPromptError(err)
	}
	preview, err := s.PreviewLoader.Load(cfg)
	if err != nil {
		s.log(traceID, SystemPromptActionUpdate, "error", err)
		return bus.ServiceResult{}, bus.WrapError(bus.ServiceErrorInternal, err)
	}
	s.log(traceID, SystemPromptActionUpdate, "success", nil)
	return bus.ResultSuccess(systemResponseFrom(files, preview)), nil
}

func (s Service) loadSystemFiles() (bridgeconfig.Config, bridgeconfig.SystemPromptFiles, error) {
	cfg, err := s.config()
	if err != nil {
		return bridgeconfig.Config{}, bridgeconfig.SystemPromptFiles{}, err
	}
	files, err := bridgeconfig.LoadSystemPromptFiles(cfg.PromptsDir)
	return cfg, files, err
}

func (s Service) config() (bridgeconfig.Config, error) {
	if s.Store == nil {
		return bridgeconfig.Config{}, errors.New("config store is not configured")
	}
	return s.Store.Config()
}

func systemResponseFrom(files bridgeconfig.SystemPromptFiles, preview Preview) SystemResponse {
	return SystemResponse{
		CorePrompt:      files.CorePrompt,
		RenderedPrompt:  preview.RenderedPrompt,
		PromptLibrary:   files.PromptLibrary,
		ToolDefinitions: preview.ToolDefinitions,
	}
}

func ToolDefinitionsFrom(defs []llm.ToolDef) []ToolDefinition {
	return toolschema.DefinitionsFrom(defs)
}

func hasSystemPromptUpdate(req bridgeconfig.SystemPromptUpdateRequest) bool {
	return req.CorePrompt != nil || req.PromptLibrary != nil
}

func mapSystemPromptError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, bridgeconfig.ErrSystemPromptUpdateEmpty) {
		return bus.WrapError(bus.ServiceErrorInvalidInput, err)
	}
	if errors.Is(err, bridgeconfig.ErrSystemPromptUpdateConflict) {
		return bus.WrapError(bus.ServiceErrorInvalidInput, err)
	}
	if errors.Is(err, bridgeconfig.ErrSystemPromptLibraryInvalid) {
		return bus.WrapError(bus.ServiceErrorInvalidInput, err)
	}
	return bus.WrapError(bus.ServiceErrorInternal, err)
}
