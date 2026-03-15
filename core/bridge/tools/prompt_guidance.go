package tools

import "strings"

func FormatPromptGuidanceForCatalog(catalog ToolCatalog) string {
	names := toolNameSet(CatalogToolNames(catalog))
	lines := []string{
		"- Use only the tools included in the structured tool schema for this turn.",
	}
	lines = append(lines, rssPromptGuidance(names)...)
	lines = append(lines, workspacePromptGuidance(names)...)
	lines = append(lines, humanPromptGuidance(names)...)
	lines = append(lines, graphQLPromptGuidance(names)...)
	lines = append(lines, screenPromptGuidance(names)...)
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
	switch {
	case names["script_exec"] && names["read_and_summarize"]:
		return []string{
			"- Use `read_and_summarize` for broad local triage, then use `script_exec` for exact reads, searches, edits, and shell/script work.",
		}
	case names["script_exec"]:
		return []string{
			"- Prefer `script_exec` for exact workspace reads, searches, edits, and shell/script work.",
		}
	case names["read_and_summarize"]:
		return []string{
			"- Use `read_and_summarize` for broad multi-file triage; rely on the available workspace tools for exact verification before changing code.",
		}
	default:
		return nil
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

func graphQLPromptGuidance(names map[string]bool) []string {
	lines := make([]string, 0, 2)
	if names["graphql_schema_lookup"] && (names["graphql_query"] || names["graphql_mutation"]) {
		lines = append(lines, "- Inspect sources, domains, and allowed policies with `graphql_schema_lookup` before GraphQL queries or writes.")
	}
	if !names["graphql_mutation"] {
		return lines
	}
	line := "- Before any GraphQL write, call `graphql_mutation(action=\"prepare\")` first and commit only after explicit user approval for the returned `intent_id`."
	if names["graphql_schema_lookup"] {
		line = "- Before any GraphQL write, inspect allowed policies with `graphql_schema_lookup`, then call `graphql_mutation(action=\"prepare\")`. Commit only after explicit user approval for the returned `intent_id`."
	}
	return append(lines, line)
}

func screenPromptGuidance(names map[string]bool) []string {
	if !names["screen_action"] {
		return nil
	}
	return []string{
		"- Prefer `screen_action.click_text` when visible labels exist; use `click_icon` only for unlabeled or template-driven targets.",
	}
}
