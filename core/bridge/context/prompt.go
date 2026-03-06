package context

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	defaultPromptVersion = "1.0"
	defaultPromptPath    = "prompts.yaml"
)

const defaultSystemPromptTemplate = `You are Ghost-OS bridge agent, an AI-driven digital twin execution layer.

## Core Capabilities
You can coordinate local execution, web retrieval, browser interaction, and human confirmation.

## Available Tools
- script_exec: Execute Python scripts in a sandbox. Use for local shell/file/search/web workflows.
- web_search: Search the public web and return concise results.
- browser_action: Execute browser-native actions (query/click/type/scroll).
- ask_human: Ask the user for a required decision or missing input.

## Operating Context
- OS: {{os_type}}
- Available tools: {{tools_count}}
- Max turns: {{max_turns}}

## Guidelines
- Choose tools only when needed. If a direct answer is enough, respond directly.
- Prefer script_exec for local system operations and code/file changes.
- When calling script_exec, always provide a non-empty "script" field. Never call it with {}.
- Use web_search for internet lookup tasks.
- Use browser_action only when browser-native interaction is required.
- Use ask_human when execution is blocked by missing user choice or confirmation.
- For normal turns, reply with plain natural text.
- Only when you intentionally end the entire session, output JSON only: {"signal":"END_SESSION","message":"<final reply>"}.
- Keep actions concise, deterministic, and traceable.`

// PromptConfig 描述 prompts.yaml 的最小结构。
type PromptConfig struct {
	Version string `json:"version"`
	System  struct {
		Default string `json:"default"`
	} `json:"system"`
}

// PromptManager 负责加载与渲染系统提示词模板。
type PromptManager struct {
	config   PromptConfig
	template string
}

// NewPromptManager 从配置文件加载提示词模板。
func NewPromptManager(configPath string) (*PromptManager, error) {
	cfgPath := strings.TrimSpace(configPath)
	if cfgPath == "" {
		cfgPath = defaultPromptPath
	}

	raw, resolvedPath, err := readPromptConfigFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("read prompts config: %w", err)
	}

	cfg, err := parsePromptYAML(string(raw))
	if err != nil {
		return nil, fmt.Errorf("parse prompts config %q: %w", resolvedPath, err)
	}

	return &PromptManager{
		config:   cfg,
		template: cfg.System.Default,
	}, nil
}

// NewPromptManagerWithDefault 使用内置模板，避免配置缺失时阻塞启动。
func NewPromptManagerWithDefault() *PromptManager {
	cfg := PromptConfig{
		Version: defaultPromptVersion,
	}
	cfg.System.Default = defaultSystemPromptTemplate

	return &PromptManager{
		config:   cfg,
		template: cfg.System.Default,
	}
}

// Render 渲染系统提示词并替换模板变量。
func (pm *PromptManager) Render(vars map[string]string) string {
	if pm == nil || strings.TrimSpace(pm.template) == "" {
		return strings.TrimSpace(RenderTemplate(defaultSystemPromptTemplate, vars))
	}

	return strings.TrimSpace(RenderTemplate(pm.template, vars))
}

func readPromptConfigFile(path string) ([]byte, string, error) {
	candidates := promptPathCandidates(path)
	var lastErr error

	for _, candidate := range candidates {
		raw, err := os.ReadFile(candidate)
		if err == nil {
			return raw, candidate, nil
		}
		lastErr = err
	}

	if lastErr == nil {
		lastErr = errors.New("prompt config path is empty")
	}
	return nil, "", fmt.Errorf("tried %v: %w", candidates, lastErr)
}

func promptPathCandidates(path string) []string {
	normalized := strings.TrimSpace(path)
	if normalized == "" {
		normalized = defaultPromptPath
	}
	if filepath.IsAbs(normalized) {
		return []string{normalized}
	}

	primary := filepath.Clean(normalized)
	fallback := filepath.Join("core", "bridge", primary)
	if fallback == primary {
		return []string{primary}
	}
	return []string{primary, fallback}
}

func parsePromptYAML(raw string) (PromptConfig, error) {
	normalized := strings.ReplaceAll(raw, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")

	cfg := PromptConfig{}
	inSystem := false
	systemIndent := 0

	for i := 0; i < len(lines); {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			i++
			continue
		}

		indent := leadingIndent(line)
		if indent == 0 {
			inSystem = false
			switch {
			case strings.HasPrefix(trimmed, "version:"):
				cfg.Version = parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "version:")))
			case trimmed == "system:":
				inSystem = true
				systemIndent = indent
			}
			i++
			continue
		}

		if !inSystem || indent <= systemIndent {
			i++
			continue
		}

		local := strings.TrimSpace(line)
		if !strings.HasPrefix(local, "default:") {
			i++
			continue
		}

		value := strings.TrimSpace(strings.TrimPrefix(local, "default:"))
		if value == "|" || value == "|+" || value == "|-" {
			blockValue, next := readYAMLBlock(lines, i+1, indent)
			cfg.System.Default = blockValue
			i = next
			continue
		}

		cfg.System.Default = parseYAMLScalar(value)
		i++
	}

	if strings.TrimSpace(cfg.Version) == "" {
		cfg.Version = defaultPromptVersion
	}
	if strings.TrimSpace(cfg.System.Default) == "" {
		return PromptConfig{}, errors.New("system.default is required")
	}

	return cfg, nil
}

func readYAMLBlock(lines []string, start int, parentIndent int) (string, int) {
	blockLines := make([]string, 0, len(lines)-start)
	i := start

	for ; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		indent := leadingIndent(line)

		if trimmed != "" && indent <= parentIndent {
			break
		}
		blockLines = append(blockLines, line)
	}

	return normalizeBlockLines(blockLines), i
}

func normalizeBlockLines(lines []string) string {
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}

	end := len(lines)
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}

	lines = lines[start:end]
	if len(lines) == 0 {
		return ""
	}

	minIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := leadingIndent(line)
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}

	if minIndent < 0 {
		return ""
	}

	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			out = append(out, "")
			continue
		}
		if len(line) < minIndent {
			out = append(out, strings.TrimSpace(line))
			continue
		}
		out = append(out, line[minIndent:])
	}

	return strings.Join(out, "\n")
}

func parseYAMLScalar(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}

	if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
		unquoted, err := strconv.Unquote(value)
		if err == nil {
			return unquoted
		}
	}

	if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") && len(value) >= 2 {
		return strings.ReplaceAll(value[1:len(value)-1], "''", "'")
	}

	return trimInlineComment(value)
}

func trimInlineComment(raw string) string {
	if idx := strings.Index(raw, " #"); idx >= 0 {
		return strings.TrimSpace(raw[:idx])
	}
	return strings.TrimSpace(raw)
}

func leadingIndent(line string) int {
	for i, ch := range line {
		if ch != ' ' && ch != '\t' {
			return i
		}
	}
	return len(line)
}
