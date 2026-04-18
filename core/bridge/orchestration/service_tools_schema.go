package orchestration

import (
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
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
	schemas := make(map[string]map[string]any, len(defs))
	for _, def := range defs {
		decoded, err := decodeToolSchema(def.Parameters, def.Name)
		if err != nil {
			return nil, err
		}
		if decoded != nil {
			schemas[def.Name] = decoded
		}
	}
	return schemas, nil
}

func decodeToolSchema(raw json.RawMessage, toolName string) (map[string]any, error) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil, nil
	}
	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("decode tool schema for %q: %w", toolName, err)
	}
	schema, ok := payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("tool %q schema must be a JSON object", toolName)
	}
	return schema, nil
}

func schemaByToolName(schemasByName map[string]map[string]any, toolName string) (map[string]any, bool) {
	if schemasByName == nil {
		return nil, false
	}
	schema, ok := schemasByName[strings.TrimSpace(toolName)]
	if !ok || schema == nil {
		return nil, false
	}
	return schema, true
}
