package orchestration

import (
	"encoding/json"

	"ghost-os/bridge/llm"
	apptools "ghost-os/bridge/orchestration/internal/app/tools"
)

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
