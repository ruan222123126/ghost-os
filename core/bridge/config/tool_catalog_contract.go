package config

// configuredToolCatalog 是配置层的稳定工具契约，避免 config 反向依赖 tools 包。
const (
	configToolNameAskHuman = "ask_human"
	configToolNameTFind    = "tfind"
)

var configuredToolCatalog = []string{
	"script_exec",
	"codex_cli",
	"web_search",
	"screen_control",
	configToolNameTFind,
	configToolNameAskHuman,
}

var workflowToolDenylist = map[string]string{
	configToolNameAskHuman: "requires an active session",
	configToolNameTFind:    "requires an active session",
}
