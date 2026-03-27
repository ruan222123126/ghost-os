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
	lines = append(lines, rssPromptGuidance(names)...)
	lines = append(lines, workspacePromptGuidance(protocol, names)...)
	lines = append(lines, memoryPromptGuidance(protocol, names)...)
	lines = append(lines, toolSearchPromptGuidance(protocol, names)...)
	lines = append(lines, humanPromptGuidance(protocol, names)...)
	lines = append(lines, webRooterPromptGuidance(names)...)
	lines = append(lines, screenPromptGuidance(names)...)
	lines = append(lines, computerUsePromptGuidance(names)...)
	return strings.Join(lines, "\n")
}

func rssPromptGuidance(names map[string]bool) []string {
	if !names["feed_manage"] && !names["rss_fetch"] {
		return nil
	}
	return []string{
		"- RSS inbox polling and AI filtering are backend system pipelines. Use visible RSS tools only for explicit feed management or feed retrieval work.",
	}
}

func workspacePromptGuidance(protocol promptGuidanceProtocol, names map[string]bool) []string {
	if !names["script_exec"] || !names["read_and_summarize"] {
		return nil
	}
	lines := []string{
		"- Use `read_and_summarize` for broad local triage, then use `script_exec` for exact reads, searches, edits, and shell/script work.",
	}
	if protocol == promptGuidanceProtocolGraphQL {
		lines = append(lines, "- Minimal `script_exec` GraphQL example: `mutation { script_exec(script: \"print(\\\"ok\\\")\") }`.")
	}
	return lines
}

func memoryPromptGuidance(protocol promptGuidanceProtocol, names map[string]bool) []string {
	if !names["memory_manage"] {
		return nil
	}
	lines := []string{
		"- Use `memory_manage` only for explicit long-term notes that should persist by stable URI.",
		"- Prefer URIs like `user://preferences/editor` or `project://roadmap/current`; use `create` for the first write.",
		"- Before `update` or `delete`, first confirm the exact URI with `read`, `list`, or `read` on `system://index`; do not guess URIs.",
		"- `system://index` and `system://recent` are read-only discovery entries.",
	}
	if protocol == promptGuidanceProtocolGraphQL {
		lines = append(
			lines,
			"- Minimal `memory_manage` create example: `mutation { memory_manage(operation: create, uri: \"user://preferences/editor\", content: \"Prefer vim keybindings\") }`.",
		)
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
			"- When you are unsure which tools are visible, start with `mutation { tfind(action: search, query: \"...\") }` to discover the smallest suitable optional tool.",
			"- After `tfind(action: load)`, the loaded tool becomes available in the same user turn on the next completion.",
			"- Minimal `tfind(action: load)` example: `mutation { tfind(action: load, tool_names: [\"browser_control\"]) }`.",
			"- Use `tfind(action: list)` only to inspect the current dynamic tool load state.",
			"- Unload tools you no longer need with `tfind(action: unload)`.",
			"- Never repeat or fabricate `[GRAPHQL_TOOL_RESULT]` in assistant text.",
		}
	}
	return []string{
		"- Use `tfind` when the currently visible tools are insufficient.",
		"- Start with `tfind` using `action=search` to find the smallest suitable optional tool.",
		"- After `tfind` with `action=load`, the loaded tool becomes available in the same user turn on the next completion.",
		"- Use `tfind` with `action=list` only to inspect the current dynamic tool load state.",
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
		lines = append(lines, "- Minimal `ask_human` options example: `mutation { ask_human(prompt: \"Which environment should I use?\", options: [{label: \"staging\"}, {label: \"Other\", allow_custom: true}]) }`.")
	}
	return lines
}

func webRooterPromptGuidance(names map[string]bool) []string {
	if !names[webRooterToolName] {
		return nil
	}
	lines := []string{
		"- Use `web_rooter` for stateless HTTP web research only. Choose exactly one supported `action`, and provide every action parameter explicitly in `params`.",
		"- Prefer `web_rooter` when the task needs citations, source attribution, multi-source cross-checking, academic material, or deeper research.",
	}
	if names["web_search"] {
		lines = append(lines, "- Use `web_search` for lighter real-time web lookups when citation-rich research is unnecessary.")
	}
	return lines
}

func screenPromptGuidance(names map[string]bool) []string {
	if !names["screen_action"] {
		return nil
	}
	return []string{
		"- Prefer `screen_action.click_text` when visible labels exist; use `click_icon` only for unlabeled or template-driven targets.",
	}
}

func computerUsePromptGuidance(names map[string]bool) []string {
	if !names[computerUseToolName] {
		return nil
	}
	lines := []string{
		"- Use `computer_use` only for desktop visual tasks that cannot be solved through scripts, APIs, or DOM/browser-native control.",
	}
	if names["browser_control"] {
		lines = append(lines, "- Prefer `browser_control` for browser tasks before using `computer_use`.")
	}
	if names["script_exec"] {
		lines = append(lines, "- Prefer `script_exec` for scriptable local operations before using `computer_use`.")
	}
	return lines
}
