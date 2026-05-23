package config

import "strings"

func joinToolPromptLines(lines ...string) string {
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

var toolPromptDefaults = map[string]string{
	"list_files":   "List the direct children of a directory inside the sandbox. Directory entries end with '/'.",
	"read_file":    "Read file text, optionally by 1-based line range. Returns line-numbered text and reads at most 200 lines per call.",
	"search_files": "Search exact text under a directory inside the sandbox. Returns stable path:line:text matches.",
	"write_file":   "Create, overwrite, or append a UTF-8 text file inside allowed paths. Missing parent directories are created automatically. `content` may be empty; `mode` is `write` or `append`.",
	"apply_diff":   "Apply a unified diff to one file inside allowed write paths.",
	"bash_exec":    "Run a shell command in the sandbox bash shell. Default is one-shot stdout output; set `interactive=true` for persistent session mode with `session_id` reuse.",
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

var toolLegacyPromptDefaults = map[string][]string{
	"script_exec": {
		"Python sandbox. Helpers injected (positional or named params, no imports): apply_diff(path, diff_text), bash_exec(cmd, max_output_chars=None), fetch_webpage(url), list_files(path), read_file(path, start, end), search_files(query, path, max), write_file(path, content, mode). list_files returns string entries with directories suffixed by /. Prefer search_files over bash_exec. No blocked builtins (open/eval/exec). Output concise JSON/text.",
		"Python sandbox. Injected `tools` helpers (use named params, no imports): apply_diff(path, diff_text), bash_exec(command, max_output_chars=None), fetch_webpage(url), list_files(path='.'), read_file(path, start_line, end_line), search_files(query, path='.', max_results=50), write_file(path, content, mode='write'). Rules: Prefer search_files over bash_exec. No blocked builtins (open/eval/exec/input). Print concise structured output (e.g., JSON).",
		joinToolPromptLines(
			"Execute a Python script in the fallback sandbox as the primary local workspace tool.",
			"",
			"Allowed helpers: tools.apply_diff(path, diff_text), tools.bash_exec(command, max_output_chars=None), tools.fetch_webpage(url), tools.list_files(path='.'), tools.read_file(path, start_line=None, end_line=None), tools.search_files(query, path='.', max_results=50), tools.write_file(path, content, mode='write').",
			"Sandbox limits are enforced by the execution layer.",
			"",
			"Use plain Python plus the injected `tools` object, and call helper methods with named parameters such as `tools.search_files(query='token', path='.')` or `tools.read_file(path='...')`.",
			"Do not use `import tools` or `from tools...`; `tools` is a runtime object, not an importable module.",
			"For text retrieval, prefer `tools.search_files` before shelling out with `tools.bash_exec`; use `tools.bash_exec` only when helpers cannot cover the workflow.",
			"For longer shell stdout, request `tools.bash_exec(command='...', max_output_chars=N)` explicitly instead of depending on the default preview budget.",
			"Do not call blocked builtins (`open`, `eval`, `exec`, `compile`, `input`); use the injected helpers instead.",
			"Print concise, structured output such as JSON so later turns can parse results reliably.",
		),
		joinToolPromptLines(
			"Execute a Python script in the fallback sandbox as the primary local workspace tool.",
			"",
			"Allowed helpers: tools.apply_diff(path, diff_text), tools.bash_exec(command, max_output_chars=None), tools.fetch_webpage(url), tools.list_files(path='.'), tools.read_file(path, start_line=None, end_line=None), tools.search_files(query, path='.', max_results=50), tools.write_file(path, content, mode='write').",
			"Sandbox limits are enforced by the execution layer.",
		),
	},
	"codex_cli": {
		joinToolPromptLines(
			"Run codex start/resume asynchronously and poll status. For status, pass the command_id (or session_id) via session_id.",
			"",
			"`op` must be one of: start, resume, status. Do not use `exec`.",
			"`prompt` is required for start and resume.",
			"`session_id` is required for resume and status.",
			"`fork` is interactive-only in Codex CLI 0.130.0 and is not available through this async tool.",
		),
		"Run codex start/resume asynchronously and poll status. For status, pass the command_id (or session_id) via session_id.",
	},
	"screen_control": {
		"Screen control entrypoint. Mode 'atomic' (direct screenshot/OCR/click) or 'agent' (goal-driven desktop execution).",
		joinToolPromptLines(
			"Unified atomic screen control entrypoint for direct screenshot/text/icon/click/input actions.",
			"",
			"Use `action` for direct screen operations; `mode=\"atomic\"` is optional.",
			"`action` must be one of: screenshot, find_text, find_icon, click_icon, mouse_position, text_input.",
			"For `action=\"click_icon\"`, providing both `params.x` and `params.y` performs a direct click and skips template matching.",
			"For `action=\"text_input\"`, set `params.text`; optional `params.submit=true` presses Enter after typing.",
		),
		"Unified atomic screen control entrypoint for direct screenshot/OCR/click/mouse/input actions. For action=text_input, provide params.text and optionally params.submit=true.",
	},
	"sfind": {
		joinToolPromptLines(
			"Find optional skills from SKILL.md, and load or unload them for this session.",
			"",
			"Use `sfind` when the currently visible tools are insufficient and you need a skill from `SKILL.md`.",
			"Do not use `sfind` for greetings, small talk, or ordinary plain-text replies when no extra capability is needed.",
			"Start with `action=search` to find the smallest suitable skill.",
			"Use `skill_names` when loading or unloading skills.",
			"After `action=load`, the loaded skill becomes available in the same user turn on the next completion.",
			"Use `action=list` only to inspect the current dynamic skill load state.",
		),
		"Find optional skills from SKILL.md, and load or unload them for this session.",
	},
	"ask_human": {
		joinToolPromptLines(
			"Pause and ask the user for required input before continuing, optionally with single-choice or multi-choice options.",
			"",
			"Use `ask_human` only when blocked on required user input.",
			"If you provide predefined choices, the final option must allow custom input.",
		),
		"Pause and ask the user for required input before continuing, optionally with single-choice or multi-choice options.",
	},
	"web_search": {
		"Search the web for current information. When both Tavily and Exa are configured, set provider explicitly so the agent can choose per query.",
	},
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

func normalizeToolPromptDefault(name string, prompt string) (string, bool) {
	trimmedName := strings.TrimSpace(name)
	trimmedPrompt := strings.TrimSpace(prompt)
	if trimmedPrompt == "" {
		return "", false
	}

	current, ok := toolBasePrompt(trimmedName)
	if !ok {
		return trimmedPrompt, false
	}
	if trimmedPrompt == current {
		return current, true
	}
	for _, legacy := range toolLegacyPromptDefaults[trimmedName] {
		if trimmedPrompt == strings.TrimSpace(legacy) {
			return current, true
		}
	}
	return trimmedPrompt, false
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
