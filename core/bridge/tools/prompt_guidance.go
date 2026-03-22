package tools

import "strings"

func FormatPromptGuidanceForCatalog(catalog ToolCatalog) string {
	names := toolNameSet(promptGuidanceToolNames(catalog))
	if len(names) == 0 {
		return ""
	}
	lines := make([]string, 0, 7)
	if preamble := promptGuidancePreamble(catalog); preamble != "" {
		lines = append(lines, preamble)
	}
	lines = append(lines, rssPromptGuidance(names)...)
	lines = append(lines, workspacePromptGuidance(names)...)
	lines = append(lines, toolSearchPromptGuidance(names)...)
	lines = append(lines, humanPromptGuidance(names)...)
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

func workspacePromptGuidance(names map[string]bool) []string {
	if !names["script_exec"] || !names["read_and_summarize"] {
		return nil
	}
	return []string{
		"- Use `read_and_summarize` for broad local triage, then use `script_exec` for exact reads, searches, edits, and shell/script work.",
	}
}

func toolSearchPromptGuidance(names map[string]bool) []string {
	if !names[ToolSearchToolName] {
		return nil
	}
	return []string{
		"- Use `tfind` when the currently visible tools are insufficient.",
		"- Start with `tfind(action=\"search\")` to find the smallest suitable optional tool.",
		"- After `tfind(action=\"load\")`, do not call the loaded tool in the same turn; it becomes available next turn.",
		"- Use `tfind(action=\"list\")` to check whether a loaded tool is pending, active, or expired.",
		"- Unload tools you no longer need with `tfind(action=\"unload\")`.",
	}
}

func humanPromptGuidance(names map[string]bool) []string {
	if !names[AskHumanToolName] {
		return nil
	}
	return []string{
		"- Use `ask_human` only when blocked on required user input. If you provide predefined choices, ensure one option allows custom input.",
	}
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
