// Package tools exposes bridge tool entrypoints and keeps heavier orchestration in
// focused internal subpackages.
//
// Layout:
//   - file_tools.go groups file discovery, read, search, and patch tools.
//   - exec_tools.go groups shell/script execution tools.
//   - set_project_root.go updates and persists the execution working directory.
//   - catalog.go groups registry, scope filtering, and selector metadata.
//   - contracts.go keeps shared interfaces, result hooks, and tool context wiring.
//   - tool_search.go exposes session-scoped discovery and dynamic loading of optional tools/skills.
//   - web/ contains web_search, web_rooter, rss_fetch, and feed_manage implementations.
//   - memory/ contains memory_manage, memory_learned_list, and memory_recall_debug implementations.
//   - screen/ contains screen_action, screen_control, and text_input implementations.
//   - graphql/ contains GraphQL text tool-call protocol parsing and runtime schema generation.
//   - contracts/ contains shared tool contracts used by root and domain subpackages.
//   - internal/tooljson, internal/toolparams, and internal/toolartifacts hold shared helper logic for tool payloads, params, and image artifacts.
//   - internal/readsummarize owns chunk reading, worker prompts, and formatting.
//   - internal/rss owns feed parsing and normalization.
//   - internal/websearch owns HTML result parsing and normalization.
package tools
