package tools

import (
	"fmt"
	"sort"
	"strings"

	"ghost-os/bridge/llm"
)

const (
	AskHumanToolName   = "ask_human"
	ToolSearchToolName = "tfind"
)

// ToolMetadata 是供轻量 selector 使用的低成本工具描述，不暴露完整 schema。
type ToolMetadata struct {
	Name      string
	Domain    string
	Tags      []string
	ShortDesc string
	AlwaysOn  bool
	OnDemand  bool
}

// Registry 按名称管理工具，并提供给 LLM 的工具定义列表。
type Registry struct {
	tools map[string]Tool
}

// ScopedCatalog 在现有工具目录上做白名单过滤，避免复制工具实现。
type ScopedCatalog struct {
	registry  ToolCatalog
	allowlist map[string]bool
}

// NewRegistry 初始化空工具注册表。
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
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
		defs = append(defs, ToolDefFromTool(r.tools[name]))
	}
	return defs
}

// NewScopedCatalog 创建一个仅暴露指定工具名的目录包装器。
func NewScopedCatalog(registry ToolCatalog, allowedTools []string) *ScopedCatalog {
	allowlist := make(map[string]bool, len(allowedTools))
	for _, name := range allowedTools {
		if name = strings.TrimSpace(name); name != "" {
			allowlist[name] = true
		}
	}
	return &ScopedCatalog{registry: registry, allowlist: allowlist}
}

func (sc *ScopedCatalog) Get(name string) Tool {
	if sc == nil || sc.registry == nil || !sc.allowlist[name] {
		return nil
	}
	return sc.registry.Get(name)
}

func (sc *ScopedCatalog) ToolDefs() []llm.ToolDef {
	if sc == nil || sc.registry == nil {
		return nil
	}

	allDefs := sc.registry.ToolDefs()
	filtered := make([]llm.ToolDef, 0, len(sc.allowlist))
	for _, def := range allDefs {
		if sc.allowlist[def.Name] {
			filtered = append(filtered, def)
		}
	}
	return filtered
}

// GetToolMetadata 返回稳定顺序的 selector 元数据清单。
func GetToolMetadata() []ToolMetadata {
	return []ToolMetadata{
		{Name: "script_exec", Domain: "sandbox", Tags: []string{"execute", "script", "complex"}, ShortDesc: "Run a Python script in sandbox."},
		{Name: "codex_cli", Domain: "sandbox", Tags: []string{"execute", "codex", "async"}, ShortDesc: "Run Codex CLI asynchronously."},
		{Name: "web_search", Domain: "web", Tags: []string{"search", "internet", "research"}, ShortDesc: "Search the web."},
		{Name: screenControlToolName, Domain: "screen", Tags: []string{"interactive", "ocr", "icon", "input", "text", "keyboard", "desktop", "gui"}, ShortDesc: "Unified screen control entrypoint."},
		{Name: ToolSearchToolName, Domain: "tools", Tags: []string{"search", "load", "unload", "catalog", "skill"}, ShortDesc: "Find or load optional tools and skills."},
		{Name: AskHumanToolName, Domain: "human", Tags: []string{"interactive", "safety"}, ShortDesc: "Ask the user when blocked."},
	}
}

// FormatMetadataForSelector 将元数据压缩为易于 LLM 读取的短文本。
func FormatMetadataForSelector() string {
	return formatMetadataLines(selectorVisibleMetadata(nil))
}

func FormatMetadataForCatalog(catalog ToolCatalog) string {
	filtered := selectorVisibleMetadata(catalog)
	if len(filtered) == 0 {
		return ""
	}
	return formatMetadataLines(filtered)
}

func formatMetadataLines(metadata []ToolMetadata) string {
	lines := make([]string, 0, len(metadata))
	for _, item := range metadata {
		line := fmt.Sprintf("%s | domain=%s | tags=%s", item.Name, item.Domain, strings.Join(item.Tags, ","))
		if item.AlwaysOn {
			line += " | always_on=true"
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
