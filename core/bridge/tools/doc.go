// Package tools exposes bridge tool entrypoints and keeps heavier orchestration in
// focused internal subpackages.
//
// Layout:
//   - file_tools.go groups file discovery, read, search, and patch tools.
//   - exec_tools.go groups shell/script execution tools.
//   - catalog.go groups registry, scope filtering, and selector metadata.
//   - contracts.go keeps shared interfaces, result hooks, and tool context wiring.
//   - tool_search.go exposes session-scoped discovery and dynamic loading of optional tools/skills.
//   - web/ contains web_search, web_rooter, image_generate, and RSS fetch helper implementations.
//   - screen/ contains screen_action and screen_control implementations.
//   - graphql/ contains GraphQL text tool-call protocol parsing and runtime schema generation.
//   - contracts/ contains shared tool contracts used by root and domain subpackages.
//   - internal/tooljson, internal/toolparams, and internal/toolartifacts hold shared helper logic for tool payloads, params, and image artifacts.
//   - internal/rss owns feed parsing and normalization.
//   - internal/websearch owns HTML result parsing and normalization.
package tools
