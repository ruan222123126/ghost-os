package tools

import "strings"

func FormatPromptGuidanceForCatalog(catalog ToolCatalog) string {
	names := toolNameSet(promptGuidanceToolNames(catalog))
	if len(names) == 0 {
		return ""
	}
	protocol := promptGuidanceProtocolForCatalog(catalog)
	lines := make([]string, 0, 7)
	if preamble := promptGuidancePreamble(catalog); preamble != "" {
		lines = append(lines, preamble)
	}
	lines = append(lines, scriptExecPromptGuidance(protocol, names)...)
	lines = append(lines, toolSearchPromptGuidance(protocol, names)...)
	lines = append(lines, humanPromptGuidance(protocol, names)...)
	lines = append(lines, codexCLIPromptGuidance(names)...)
	lines = append(lines, screenControlPromptGuidance(names)...)
	lines = append(lines, screenPromptGuidance(names)...)
	return strings.Join(lines, "\n")
}

func scriptExecPromptGuidance(protocol promptGuidanceProtocol, names map[string]bool) []string {
	if !names["script_exec"] {
		return nil
	}
	lines := []string{
		"- In `script_exec`, use plain Python plus the injected `tools` object, and call helper methods with named parameters (for example `tools.search_files(query='token', path='.')` or `tools.read_file(path='...')`).",
		"- Do not use `import tools` or `from tools...`; `tools` is a runtime object, not an importable module.",
		"- For text retrieval, prefer `tools.search_files` before shelling out with `tools.bash_exec`; use `tools.bash_exec` only when helpers cannot cover the workflow.",
		"- Do not call blocked builtins (`open`, `eval`, `exec`, `compile`, `input`); use `tools.read_file`, `tools.write_file`, `tools.apply_diff`, `tools.search_files`, or `tools.bash_exec` instead.",
		"- Print concise, structured output (for example JSON) so later turns can parse results reliably.",
	}
	if protocol == promptGuidanceProtocolGraphQL {
		lines = append(lines, "- Minimal `script_exec` tag example: `<t:ID>{\"script\":\"print(\\\"ok\\\")\"}</t>` (replace `ID` with the listed tool id).")
	}
	return lines
}

func toolSearchPromptGuidance(protocol promptGuidanceProtocol, names map[string]bool) []string {
	if !names[ToolSearchToolName] {
		return nil
	}
	if protocol == promptGuidanceProtocolGraphQL {
		return []string{
			"- Use `tfind` when the currently visible tools are insufficient.",
			"- Do not use `tfind` for greetings, small talk, or ordinary plain-text replies when no extra capability is needed.",
			"- When you are unsure which tools are visible, start with the listed `tfind` id and call `<t:ID>{\"action\":\"search\",\"query\":\"...\"}</t>`.",
			"- Search skills with `kind=\"skill\"`, for example: `<t:ID>{\"action\":\"search\",\"kind\":\"skill\",\"query\":\"release\"}</t>`.",
			"- When calling tools, keep the assistant message focused on tool tags and avoid extra wrappers.",
			"- After `tfind(action: load)`, the loaded tool becomes available in the same user turn on the next completion.",
			"- Load skills with `kind=\"skill\"`; skill dependencies can auto-load required tools.",
			"- Minimal `tfind(action: load)` tag example: `<t:ID>{\"action\":\"load\",\"tool_names\":[\"web_search\"]}</t>`.",
			"- Use `tfind(action: list)` only to inspect the current dynamic tool/skill load state.",
			"- Unload tools you no longer need with `tfind(action: unload)`.",
			"- Never repeat or fabricate `[TOOL_TAG_RESULT]` in assistant text.",
		}
	}
	return []string{
		"- Use `tfind` when the currently visible tools are insufficient.",
		"- Do not use `tfind` for greetings, small talk, or ordinary plain-text replies when no extra capability is needed.",
		"- Start with `tfind` using `action=search` to find the smallest suitable optional tool.",
		"- Use `kind=skill` when searching or loading skills from SKILL.md.",
		"- After `tfind` with `action=load`, the loaded tool becomes available in the same user turn on the next completion.",
		"- Use `tfind` with `action=list` only to inspect the current dynamic tool/skill load state.",
		"- Unload tools you no longer need with `tfind` using `action=unload`.",
	}
}

func humanPromptGuidance(protocol promptGuidanceProtocol, names map[string]bool) []string {
	if !names[AskHumanToolName] {
		return nil
	}
	lines := []string{
		"- Use `ask_human` only when blocked on required user input. If you provide predefined choices, the final option must allow custom input.",
	}
	if protocol == promptGuidanceProtocolGraphQL {
		lines = append(lines, "- Minimal `ask_human` tag example: `<t:ID>{\"prompt\":\"Which environment should I use?\",\"options\":[{\"label\":\"staging\"},{\"label\":\"Other\",\"allow_custom\":true}]}</t>`.")
	}
	return lines
}

func codexCLIPromptGuidance(names map[string]bool) []string {
	if !names["codex_cli"] {
		return nil
	}
	return []string{
		"- `codex_cli` `op` must be one of: start, resume, fork, status (not `exec`).",
		"- `codex_cli` requires `prompt` for start/resume/fork and `session_id` for resume/fork/status.",
	}
}

func screenControlPromptGuidance(names map[string]bool) []string {
	if !names[screenControlToolName] {
		return nil
	}
	return []string{
		"- `screen_control` is the unified atomic screen entrypoint; use `action` (optionally with `mode=\"atomic\"`) for direct screen operations.",
		"- `screen_control` atomic action must be one of: screenshot, ocr_scan, click_text, find_icon, click_icon, mouse_position, text_input.",
		"- For `screen_control` `action=\"click_icon\"`, providing both `params.x` and `params.y` performs a direct click and skips template matching.",
		"- For `screen_control` `action=\"text_input\"`, set `params.text`; optional `params.submit=true` presses Enter after typing.",
	}
}

func screenPromptGuidance(names map[string]bool) []string {
	if !names["screen_action"] {
		return nil
	}
	return []string{
		"- `screen_action` action must be one of: screenshot, ocr_scan, click_text, find_icon, click_icon.",
		"- Prefer `screen_action.click_text` when visible labels exist; use `click_icon` only for unlabeled or template-driven targets.",
		"- For `screen_action.click_icon`, providing both `x` and `y` performs a direct click and skips template matching.",
	}
}
