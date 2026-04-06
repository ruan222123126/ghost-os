package config

// configuredToolCatalog 是配置层的稳定工具契约，避免 config 反向依赖 tools 包。
const (
	configToolNameAskHuman = "ask_human"
	configToolNameTFind    = "tfind"
)

var configuredToolCatalog = []string{
	"read_and_summarize",
	"send_file",
	"set_project_root",
	"script_exec",
	"codex_cli",
	"web_search",
	"web_rooter",
	"feed_manage",
	"rss_fetch",
	"memory_manage",
	"memory_learned_list",
	"memory_recall_debug",
	"screen_control",
	"text_input",
	"task_manage",
	configToolNameTFind,
	configToolNameAskHuman,
}

var workflowToolDenylist = map[string]string{
	configToolNameAskHuman: "requires an active session",
	configToolNameTFind:    "requires an active session",
	"send_file":            "requires an active session",
	"computer_use":         "requires an active session",
}
