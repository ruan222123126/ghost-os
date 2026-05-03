package config

// configuredToolCatalog 是配置层的稳定工具契约，避免 config 反向依赖 tools 包。
const (
	configToolNameAskHuman = "ask_human"
	configToolNameSFind    = "sfind"
)

var configuredToolCatalog = []string{
	"list_files",
	"read_file",
	"search_files",
	"write_file",
	"apply_diff",
	"bash_exec",
	"script_exec",
	"codex_cli",
	"web_search",
	"screen_control",
	configToolNameSFind,
	configToolNameAskHuman,
}

var workflowToolDenylist = map[string]string{
	configToolNameAskHuman: "requires an active session",
	configToolNameSFind:    "requires an active session",
}
