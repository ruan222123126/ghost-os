package tools

import "ghost-os/bridge/llm"

type structuredToolHiddenCatalog struct {
	base ToolCatalog
}

type graphQLToolDefSource interface {
	GraphQLToolDefs() []llm.ToolDef
}

func NewStructuredToolHiddenCatalog(base ToolCatalog) ToolCatalog {
	if base == nil {
		return nil
	}
	return structuredToolHiddenCatalog{base: base}
}

func (c structuredToolHiddenCatalog) Get(name string) Tool {
	if c.base == nil {
		return nil
	}
	return c.base.Get(name)
}

func (structuredToolHiddenCatalog) PromptGuidancePreamble() string {
	return "- Use only the tools exposed in the current tool id list for this turn."
}

func (structuredToolHiddenCatalog) PromptGuidanceProtocol() promptGuidanceProtocol {
	return promptGuidanceProtocolGraphQL
}

func (c structuredToolHiddenCatalog) PromptGuidanceToolNames() []string {
	if c.base == nil {
		return nil
	}
	return toolDefNames(c.base.ToolDefs())
}

func (c structuredToolHiddenCatalog) GraphQLToolDefs() []llm.ToolDef {
	if c.base == nil {
		return nil
	}
	return c.base.ToolDefs()
}

func (structuredToolHiddenCatalog) ToolDefs() []llm.ToolDef {
	return nil
}
