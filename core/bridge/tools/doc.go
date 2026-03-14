// Package tools exposes bridge tool entrypoints and keeps heavier orchestration in
// focused internal subpackages.
//
// Layout:
//   - file_tools.go groups file discovery, read, search, and patch tools.
//   - exec_tools.go groups shell/script execution tools.
//   - set_project_root.go updates and persists the execution working directory.
//   - feed_tools.go provides shared RSS source management tools.
//   - rss_fetch.go provides RSS/Atom retrieval and normalization.
//   - catalog.go groups registry, scope filtering, and selector metadata.
//   - contracts.go keeps shared interfaces, result hooks, and tool context wiring.
//   - screen_action*.go splits screen screenshot, OCR, matching, and click flows by domain.
//   - browser_control*.go splits browser session, endpoint discovery, DOM actions, and screenshots by domain.
//   - text_input.go exposes focused-field text entry.
//   - internal/tooljson, internal/toolparams, and internal/toolartifacts hold shared helper logic for tool payloads, params, and image artifacts.
//   - internal/readsummarize owns chunk reading, worker prompts, and formatting.
//   - internal/rss owns feed parsing and normalization.
//   - internal/websearch owns HTML result parsing and normalization.
package tools
