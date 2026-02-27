package tools

import (
	"fmt"
	"sort"

	"ghost-os/bridge/llm"
)

// Registry 按名称管理工具，并提供给 LLM 的工具定义列表。
type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register 只允许非空且不重复的工具名，启动期尽早失败。
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

// Get 按名称查找工具；不存在返回 nil。
func (r *Registry) Get(name string) Tool {
	return r.tools[name]
}

// ToolDefs 返回稳定排序后的工具定义，减少模型上下文抖动。
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
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
		})
	}
	return defs
}
