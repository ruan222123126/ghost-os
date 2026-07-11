package runtime

import (
	"strings"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

type sessionTurnCatalog struct {
	base      tools.ToolCatalog
	static    []string
	sess      *session.Session
	idleTurns int
	selector  bool
}

func newSessionTurnCatalog(base tools.ToolCatalog, static []string, sess *session.Session, idleTurns int, selector bool) tools.ToolCatalog {
	if base == nil {
		return nil
	}
	return &sessionTurnCatalog{
		base:      base,
		static:    normalizeToolNames(static),
		sess:      sess,
		idleTurns: idleTurns,
		selector:  selector,
	}
}

func (c *sessionTurnCatalog) Get(name string) tools.Tool {
	if c == nil || c.base == nil || !toolNameSet(c.allowedToolNames())[strings.TrimSpace(name)] {
		return nil
	}
	return c.base.Get(strings.TrimSpace(name))
}

func (c *sessionTurnCatalog) ToolDefs() []llm.ToolDef {
	if c == nil || c.base == nil {
		return nil
	}

	allowed := toolNameSet(c.allowedToolNames())
	defs := c.base.ToolDefs()
	filtered := make([]llm.ToolDef, 0, len(defs))
	for _, def := range defs {
		if !allowed[strings.TrimSpace(def.Name)] {
			continue
		}
		filtered = append(filtered, def)
	}
	return filtered
}

func (c *sessionTurnCatalog) allowedToolNames() []string {
	if c == nil || c.base == nil {
		return nil
	}

	names := append([]string(nil), c.static...)
	if c.sess != nil {
		names = append(names, c.sess.VisibleDynamicToolNames(c.idleTurns)...)
	}
	if c.selector {
		return tools.SelectorVisibleToolNames(names)
	}
	return normalizeToolNames(names)
}
