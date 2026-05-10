package toolschema

import (
	"encoding/json"

	"ghost-os/bridge/llm"
)

type Definition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

func DefinitionsFrom(defs []llm.ToolDef) []Definition {
	if len(defs) == 0 {
		return nil
	}
	items := make([]Definition, 0, len(defs))
	for _, def := range defs {
		items = append(items, Definition{
			Name:        def.Name,
			Description: def.Description,
			Parameters:  def.Parameters,
		})
	}
	return items
}
