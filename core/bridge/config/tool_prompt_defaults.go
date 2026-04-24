package config

import "strings"

var toolPromptDefaults = map[string]string{
	"script_exec": "Execute a Python script in the fallback sandbox as the primary local workspace tool.\n\n" +
		"Allowed helpers: tools.apply_diff(path, diff_text), tools.bash_exec(command), tools.fetch_webpage(url), tools.list_files(path='.'), tools.read_file(path, start_line=None, end_line=None), tools.search_files(keyword, dir_path='.', case_sensitive=True), tools.write_file(path, content, mode='write').\n" +
		"Sandbox limits are enforced by the execution layer.",
	"codex_cli":      "Run codex start/resume/fork asynchronously and poll status. For status, pass the command_id (or session_id) via session_id.",
	"image_generate": "Generate one or more images from a text prompt through the OpenAI-compatible Images API and persist each output as a session artifact.",
	"web_search":     "Search the web for current information. When both Tavily and Exa are configured, set provider explicitly so the agent can choose per query.",
	"web_rooter":     "Call the pinned web-rooter v0.2.4 HTTP service for stateless internet research, academic search, fetch, and extract operations. Use exactly one action with an explicit params object.",
	"screen_control": "Unified atomic screen control entrypoint for direct screenshot/OCR/click actions.",
	"text_input":     "Type text into the currently focused input field. Use only after the target field is focused; set submit=true to press Enter after typing.",
	"task_manage":    "Manage scheduled agent_message tasks: create, update, list, get, or delete. session_id is optional; when omitted, runs start a new conversation.",
	"tfind":          "Find optional tools or skills, and load or unload them for this session.",
	"ask_human":      "Pause and ask the user for required input before continuing, optionally with single-choice or multi-choice options.",
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
