package config

import "strings"

var toolPromptDefaults = map[string]string{
	"script_exec": "Execute a Python script in the fallback sandbox as the primary local workspace tool.\n\n" +
		"Allowed helpers: tools.apply_diff(path, diff_text), tools.bash_exec(command), tools.fetch_webpage(url), tools.list_files(path='.'), tools.read_file(path, start_line=None, end_line=None), tools.search_files(query, path='.', max_results=50), tools.write_file(path, content, mode='write').\n" +
		"Sandbox limits are enforced by the execution layer.",
	"codex_cli":      "Run codex start/resume/fork asynchronously and poll status. For status, pass the command_id (or session_id) via session_id.",
	"web_search":     "Search the web for current information. When both Tavily and Exa are configured, set provider explicitly so the agent can choose per query.",
	"screen_control": "Unified atomic screen control entrypoint for direct screenshot/OCR/click/mouse/input actions. For action=text_input, provide params.text and optionally params.submit=true.",
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
