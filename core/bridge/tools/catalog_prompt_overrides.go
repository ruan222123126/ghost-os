package tools

import (
	"strings"

	"ghost-os/bridge/llm"
)

type promptOverrideCatalog struct {
	base      ToolCatalog
	overrides map[string]string
}

func NewPromptOverrideCatalog(base ToolCatalog, overrides map[string]string) ToolCatalog {
	if base == nil {
		return nil
	}
	normalized := normalizePromptOverrides(overrides)
	if len(normalized) == 0 {
		return base
	}
	return promptOverrideCatalog{
		base:      base,
		overrides: normalized,
	}
}

func (c promptOverrideCatalog) Get(name string) Tool {
	if c.base == nil {
		return nil
	}
	return c.base.Get(strings.TrimSpace(name))
}

func (c promptOverrideCatalog) ToolDefs() []llm.ToolDef {
	if c.base == nil {
		return nil
	}
	baseDefs := c.base.ToolDefs()
	if len(baseDefs) == 0 {
		return nil
	}

	out := make([]llm.ToolDef, 0, len(baseDefs))
	for _, def := range baseDefs {
		next := def
		if override, ok := c.overrides[strings.TrimSpace(def.Name)]; ok {
			next.Description = override
		}
		out = append(out, next)
	}
	return out
}

func normalizePromptOverrides(raw map[string]string) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]string, len(raw))
	for key, value := range raw {
		name := strings.TrimSpace(key)
		prompt := strings.TrimSpace(value)
		if name == "" || prompt == "" {
			continue
		}
		out[name] = prompt
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
