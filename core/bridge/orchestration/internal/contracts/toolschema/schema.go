package toolschema

import (
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
)

func Collect(defs []llm.ToolDef) (map[string]map[string]any, error) {
	schemas := make(map[string]map[string]any, len(defs))
	for _, def := range defs {
		decoded, err := Decode(def.Parameters, def.Name)
		if err != nil {
			return nil, err
		}
		if decoded != nil {
			schemas[def.Name] = decoded
		}
	}
	return schemas, nil
}

func Decode(raw json.RawMessage, toolName string) (map[string]any, error) {
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

func ByToolName(schemas map[string]map[string]any, toolName string) (map[string]any, bool) {
	if schemas == nil {
		return nil, false
	}
	schema, ok := schemas[strings.TrimSpace(toolName)]
	if !ok || schema == nil {
		return nil, false
	}
	return schema, true
}
