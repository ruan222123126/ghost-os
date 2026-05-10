package orchestration

import (
	"sort"
	"strings"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

type relayModeCatalog struct {
	base   tools.ToolCatalog
	extra  map[string]tools.Tool
	hidden map[string]bool
}

func newRelayModeCatalog(base tools.ToolCatalog, allowComplete bool) tools.ToolCatalog {
	extra := map[string]tools.Tool{
		"relay_update_record": tools.NewRelayUpdateRecordTool(),
	}
	if allowComplete {
		extra["relay_complete"] = tools.NewRelayCompleteTool()
	}
	return &relayModeCatalog{
		base:  base,
		extra: extra,
		hidden: map[string]bool{
			"ask_human": true,
		},
	}
}

func (c *relayModeCatalog) Get(name string) tools.Tool {
	if c == nil {
		return nil
	}
	trimmed := strings.TrimSpace(name)
	if c.hidden[trimmed] {
		return nil
	}
	if tool, ok := c.extra[trimmed]; ok {
		return tool
	}
	if c.base == nil {
		return nil
	}
	return c.base.Get(trimmed)
}

func (c *relayModeCatalog) ToolDefs() []llm.ToolDef {
	if c == nil {
		return nil
	}
	defs := relayVisibleToolDefs(c.base, c.hidden)
	for _, name := range sortedRelayToolNames(c.extra) {
		defs = append(defs, tools.ToolDefFromTool(c.extra[name]))
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].Name < defs[j].Name })
	return defs
}

func relayVisibleToolDefs(base tools.ToolCatalog, hidden map[string]bool) []llm.ToolDef {
	if base == nil {
		return nil
	}
	defs := make([]llm.ToolDef, 0)
	for _, def := range base.ToolDefs() {
		if !hidden[strings.TrimSpace(def.Name)] {
			defs = append(defs, def)
		}
	}
	return defs
}

func sortedRelayToolNames(extra map[string]tools.Tool) []string {
	names := make([]string, 0, len(extra))
	for name := range extra {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
