package tools

import (
	"sort"
	"strings"

	"ghost-os/bridge/llm"
)

type promptGuidanceCatalog interface {
	PromptGuidanceProtocol() promptGuidanceProtocol
	PromptGuidancePreamble() string
	PromptGuidanceToolNames() []string
}

type promptGuidanceProtocol string

const (
	promptGuidanceProtocolToolCalls promptGuidanceProtocol = "tool_calls"
	promptGuidanceProtocolGraphQL   promptGuidanceProtocol = "graphql"
)

func promptGuidanceProtocolForCatalog(catalog ToolCatalog) promptGuidanceProtocol {
	if catalog == nil {
		return promptGuidanceProtocolToolCalls
	}
	guidanceCatalog, ok := catalog.(promptGuidanceCatalog)
	if !ok {
		return promptGuidanceProtocolToolCalls
	}
	protocol := guidanceCatalog.PromptGuidanceProtocol()
	if protocol == "" {
		return promptGuidanceProtocolToolCalls
	}
	return protocol
}

func promptGuidancePreamble(catalog ToolCatalog) string {
	if catalog == nil {
		return ""
	}
	if guidanceCatalog, ok := catalog.(promptGuidanceCatalog); ok {
		return strings.TrimSpace(guidanceCatalog.PromptGuidancePreamble())
	}
	return "- Use only the tools included in the structured tool schema for this turn."
}

func promptGuidanceToolNames(catalog ToolCatalog) []string {
	if catalog == nil {
		return nil
	}
	if guidanceCatalog, ok := catalog.(promptGuidanceCatalog); ok {
		return normalizeVisibleToolNames(guidanceCatalog.PromptGuidanceToolNames())
	}
	return CatalogToolNames(catalog)
}

func toolDefNames(defs []llm.ToolDef) []string {
	if len(defs) == 0 {
		return nil
	}
	names := make([]string, 0, len(defs))
	for _, def := range defs {
		name := strings.TrimSpace(def.Name)
		if name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return nil
	}
	sort.Strings(names)
	return normalizeVisibleToolNames(names)
}
