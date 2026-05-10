package orchestration

import (
	"encoding/json"
	"fmt"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/orchestration/internal/contracts/toolschema"
)

func (s *bridgeService) listToolInputSchemas() (map[string]map[string]any, error) {
	if s == nil || s.runtimeFactory == nil {
		return map[string]map[string]any{}, nil
	}
	deps, err := s.runtimeFactory.Build(s.configStore)
	if err != nil {
		return nil, fmt.Errorf("build runtime dependencies for tool schemas: %w", err)
	}
	defer deps.Close()
	if deps.registry == nil {
		return map[string]map[string]any{}, nil
	}
	return collectToolSchemas(deps.registry.ToolDefs())
}

func collectToolSchemas(defs []llm.ToolDef) (map[string]map[string]any, error) {
	return toolschema.Collect(defs)
}

func decodeToolSchema(raw json.RawMessage, toolName string) (map[string]any, error) {
	return toolschema.Decode(raw, toolName)
}

func schemaByToolName(schemasByName map[string]map[string]any, toolName string) (map[string]any, bool) {
	return toolschema.ByToolName(schemasByName, toolName)
}
