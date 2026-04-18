package config

import "strings"

var toolPromptDefaults = map[string]string{
	"read_and_summarize": "Read multiple local files and summarize them with a worker model for fast triage. Use the available workspace tools to verify exact code before editing.",
	"send_file":          "Send a local file back to the current session as a downloadable attachment.",
	"set_project_root":   "Set and persist the project root directory used by execution tools. The path must exist, be a directory, and be inside the allowed read/write roots.",
	"script_exec": "Execute a Python script in the fallback sandbox as the primary local workspace tool.\n\n" +
		"Allowed helpers: tools.apply_diff(path, diff_text), tools.bash_exec(command), tools.fetch_webpage(url), tools.list_files(path='.'), tools.read_file(path, start_line=None, end_line=None), tools.search_files(keyword, dir_path='.', case_sensitive=True), tools.write_file(path, content, mode='write').\n" +
		"Sandbox limits are enforced by the execution layer.",
	"codex_cli":           "Run codex start/resume/fork asynchronously and poll status. For status, pass the command_id (or session_id) via session_id.",
	"web_search":          "Search the web for current information. When both Tavily and Exa are configured, set provider explicitly so the agent can choose per query.",
	"web_rooter":          "Call the pinned web-rooter v0.2.4 HTTP service for stateless internet research, academic search, fetch, and extract operations. Use exactly one action with an explicit params object.",
	"feed_manage":         "Manage shared RSS/Atom feed sources: subscribe, list, update metadata, or unsubscribe.",
	"rss_fetch":           "Fetch an HTTPS RSS or Atom feed and return normalized items sorted by newest first.",
	"memory_manage":       "Manage explicit persistent memory entries by stable URI using create, read, update, delete, list, and search. Use create for first write, use read/list or system://index/system://recent to discover exact URIs, and update/delete only after the target URI already exists.",
	"memory_learned_list": "Read-only listing of event-scoped learned memories with optional event_id, session_id, type, status, and query filters.",
	"memory_recall_debug": "Read-only debug view for automatic memory recall hits, ranking reasons, and the injected prompt block.",
	"screen_control":      "Unified atomic screen control entrypoint for direct screenshot/OCR/click actions.",
	"text_input":          "Type text into the currently focused input field. Use only after the target field is focused; set submit=true to press Enter after typing.",
	"task_manage":         "Manage scheduled agent_message tasks: create, update, list, get, or delete. session_id is optional; when omitted, runs start a new conversation.",
	"tfind":               "Find optional tools or skills, and load or unload them for this session.",
	"ask_human":           "Pause and ask the user for required input before continuing, optionally with single-choice or multi-choice options.",
}

func toolBasePrompt(name string) (string, bool) {
	prompt, ok := toolPromptDefaults[strings.TrimSpace(name)]
	if !ok {
		return "", false
	}
	trimmed := strings.TrimSpace(prompt)
	if trimmed == "" {
		return "", false
	}
	return trimmed, true
}

func toolBasePrompts() map[string]string {
	out := make(map[string]string, len(configuredToolCatalog))
	for _, name := range configuredToolNames() {
		prompt, ok := toolBasePrompt(name)
		if !ok {
			continue
		}
		out[name] = prompt
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
