package tools

import "strings"

const ScriptExecToolName = "script_exec"

func joinToolPromptLines(lines ...string) string {
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

var toolPromptDefaults = map[string]string{
	"list_files":   "List the direct children of a directory inside the sandbox. Directory entries end with '/'.",
	"read_file":    "Read file text, optionally by 1-based line range. Returns line-numbered text and reads at most 200 lines per call.",
	"search_files": "Search exact text under a directory inside the sandbox. Returns stable path:line:text snippets.",
	"write_file":   "Create, overwrite, or append a UTF-8 text file inside allowed paths. Missing parent directories are created automatically. `content` may be empty; `mode` is `write` or `append`.",
	"apply_diff":   "Apply a unified diff to one file inside allowed write paths.",
	"bash_exec": joinToolPromptLines(
		"Run a bash shell command in the sandbox.",
		"",
		"Default is one-shot (stateless) execution: use it for single commands that do not need to share state across calls.",
		"For one-shot commands, pass only `command` plus optional `login`, `timeout_ms`, or `max_output_chars`; do not pass `session_id`, `tty`, or `yield_time_ms`.",
		"Use `interactive=true` only when you need a persistent shell session and must reuse state across multiple calls (e.g., `cd`, `export`, multi-step scripts).",
		"",
		"Interactive rules:",
		"- To create a session: call with `interactive=true` and omit `session_id`; use the returned `session_id` on subsequent calls.",
		"- To reuse a session: call with `interactive=true` and pass the previous `session_id`.",
		"- Do NOT pass `timeout_ms` or `login` when `interactive=true`.",
		"- Do NOT set `tty=true` (not supported yet).",
	),
	"script_exec": joinToolPromptLines(
		"Run a Python script in the sandbox as a fresh one-shot execution.",
		"",
		"Top-level helpers are available without import, and `tools.*` is not supported.",
		"Top-level helpers accept positional or named parameters.",
		"Allowed helpers: list_files(path='.'), read_file(path, start_line=None, end_line=None), search_files(query, path='.', max_results=50), write_file(path, content, mode='write'), apply_diff(path, diff_text), bash_exec(command, max_output_chars=None), fetch_webpage(url).",
		"`list_files` returns a string array, and directory entries end with `/`.",
		"`open(path, mode)` is available only for UTF-8 text `r`/`w`/`a`; for retrieval or search, prefer `read_file` and `search_files`.",
		"Each call runs in a fresh process; re-import modules and recreate variables every time.",
		"Common modules `json`, `os`, and `sys` are preloaded in each fresh call.",
		"`subprocess.run/check_output` and `os.popen/os.system` are compatibility shims that route through the sandbox shell path; non-whitelisted modules such as `pathlib` remain unavailable.",
		"Prefer `search_files` over `bash_exec` for text discovery.",
		"For longer shell stdout, request `bash_exec(..., max_output_chars=N)` explicitly instead of relying on the default preview budget.",
		"Blocked builtins remain unavailable: `eval`, `exec`, `compile`, `input`.",
		"Print concise structured output such as JSON when possible.",
	),
	"codex_cli":      "Async codex runner. Rules: 'prompt' required for start/resume. 'session_id' required for resume/status (pass command_id here for status). Omit model/sandbox to use local Codex config. Fork is interactive-only in Codex CLI 0.130.0. DO NOT use 'exec'.",
	"web_search":     "Search the web for current information.",
	"screen_control": "Screen control. Mode 'atomic' (screenshot/OCR/click) or 'agent' (goal-driven execution).",
	"sfind":          "Manage dynamic skills from SKILL.md. 'search' finds them, 'load' applies them immediately for this session, 'unload' removes them, 'list' shows current state. Use ONLY when visible tools are insufficient.",
	"ask_human":      "Block and ask user for input. If 'options' are provided, the final option MUST set allow_custom=true.",
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

func ToolBasePrompt(name string) (string, bool) {
	return toolBasePrompt(name)
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
