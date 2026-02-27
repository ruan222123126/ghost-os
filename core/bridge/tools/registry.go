package tools

import (
	"fmt"
	"sort"

	"ghost-os/bridge/llm"
)

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

func (r *Registry) Register(t Tool) {
	if t == nil {
		panic("tool is nil")
	}

	name := t.Name()
	if name == "" {
		panic("tool name is empty")
	}
	if _, exists := r.tools[name]; exists {
		panic(fmt.Sprintf("duplicate tool registration: %s", name))
	}
	r.tools[name] = t
}

func (r *Registry) Get(name string) Tool {
	return r.tools[name]
}

func (r *Registry) ToolDefs() []llm.ToolDef {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)

	defs := make([]llm.ToolDef, 0, len(names))
	for _, name := range names {
		t := r.tools[name]
		defs = append(defs, llm.ToolDef{
			Type: "function",
			Function: llm.FunctionSpec{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Parameters(),
			},
		})
	}
	return defs
}
