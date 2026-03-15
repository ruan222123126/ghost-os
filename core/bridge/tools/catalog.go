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
		tool := r.tools[name]
		defs = append(defs, llm.ToolDef{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.Parameters(),
		})
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
		{Name: "read_and_summarize", Domain: "file", Tags: []string{"read", "summarize", "batch", "triage"}, ShortDesc: "Read many files and summarize."},
		{Name: "send_file", Domain: "file", Tags: []string{"export", "download", "artifact"}, ShortDesc: "Export a file artifact."},
		{Name: "set_project_root", Domain: "workspace", Tags: []string{"root", "workspace", "config"}, ShortDesc: "Set the workspace root."},
		{Name: "script_exec", Domain: "sandbox", Tags: []string{"execute", "script", "complex"}, ShortDesc: "Run a Python script in sandbox."},
		{Name: "codex_cli", Domain: "sandbox", Tags: []string{"execute", "codex", "async"}, ShortDesc: "Run Codex CLI asynchronously."},
		{Name: "web_search", Domain: "web", Tags: []string{"search", "internet", "research"}, ShortDesc: "Search the web."},
		{Name: "graphql_query", Domain: "data", Tags: []string{"graphql", "query", "read", "structured"}, ShortDesc: "Read-only structured data query.", OnDemand: true},
		{Name: "graphql_schema_lookup", Domain: "data", Tags: []string{"graphql", "schema", "inspect", "read"}, ShortDesc: "Inspect local GraphQL schema snapshot.", OnDemand: true},
		{Name: "graphql_mutation", Domain: "data", Tags: []string{"graphql", "mutation", "write", "approval"}, ShortDesc: "Prepare and commit approved GraphQL writes.", OnDemand: true},
		{Name: "feed_manage", Domain: "web", Tags: []string{"feed", "rss", "manage", "crud"}, ShortDesc: "Manage RSS/Atom sources."},
		{Name: "rss_fetch", Domain: "web", Tags: []string{"feed", "rss", "atom", "updates"}, ShortDesc: "Fetch one RSS/Atom feed."},
		{Name: "memory_manage", Domain: "memory", Tags: []string{"memory", "store", "recall", "crud"}, ShortDesc: "CRUD explicit memory."},
		{Name: "memory_learned_list", Domain: "memory", Tags: []string{"memory", "learned", "read", "debug"}, ShortDesc: "List learned memory."},
		{Name: "memory_recall_debug", Domain: "memory", Tags: []string{"memory", "recall", "debug", "read"}, ShortDesc: "Inspect memory recall."},
		{Name: "screen_action", Domain: "screen", Tags: []string{"interactive", "ocr", "icon", "native"}, ShortDesc: "Use OCR or click on screen."},
		{Name: "browser_control", Domain: "browser", Tags: []string{"browser", "automation", "web", "dom"}, ShortDesc: "Control a browser tab."},
		{Name: "text_input", Domain: "screen", Tags: []string{"input", "text", "keyboard", "native"}, ShortDesc: "Type into the focused field."},
		{Name: "task_manage", Domain: "task", Tags: []string{"schedule", "manage", "automation"}, ShortDesc: "Manage scheduled tasks."},
		{Name: ToolSearchToolName, Domain: "tools", Tags: []string{"search", "load", "unload", "catalog"}, ShortDesc: "Find or load optional tools."},
		{Name: AskHumanToolName, Domain: "human", Tags: []string{"interactive", "safety"}, ShortDesc: "Ask the user when blocked.", AlwaysOn: true},
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
