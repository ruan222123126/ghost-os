// Package tools exposes bridge tool entrypoints and keeps heavier orchestration in
// focused internal subpackages.
//
// Layout:
//   - file_tools.go groups file discovery, read, search, and patch tools.
//   - exec_tools.go groups shell/script execution tools.
//   - feed_tools.go provides shared RSS source management tools.
//   - rss_fetch.go provides RSS/Atom retrieval and normalization.
//   - catalog.go groups registry, scope filtering, and selector metadata.
//   - contracts.go keeps shared interfaces, result hooks, and tool context wiring.
//   - screen_action.go exposes screen screenshot, OCR, and template-match actions.
//   - browser_control.go exposes browser CDP navigation and DOM actions.
//   - text_input.go exposes focused-field text entry.
//   - internal/readsummarize owns chunk reading, worker prompts, and formatting.
//   - internal/rss owns feed parsing and normalization.
//   - internal/websearch owns HTML result parsing and normalization.
package tools
