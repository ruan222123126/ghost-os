package context

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

// PromptLoadOptions 描述提示词加载的可选参数。
type PromptLoadOptions struct {
	ConfigPath             string
	CoreDir                string
	CoreFiles              []string
	RuntimeConstraintFiles []string
	ResponseRuleFiles      []string
}

type promptSectionOptions struct {
	coreJob            bool
	runtimeConstraints bool
	responseRules      bool
}

type promptSectionRequirement struct {
	field       string
	placeholder string
	value       string
	optional    bool
}

func loadPromptConfigFromOptions(options PromptLoadOptions) (PromptConfig, error) {
	cfgPath := strings.TrimSpace(options.ConfigPath)
	if cfgPath == "" {
		cfgPath = defaultPromptPath
	}

	raw, resolvedPath, err := readPromptConfigFile(cfgPath)
	if err != nil {
		return PromptConfig{}, fmt.Errorf("read prompts config: %w", err)
	}

	cfg, err := parsePromptYAML(raw, promptSectionOptions{
		coreJob:            len(options.CoreFiles) > 0,
		runtimeConstraints: len(options.RuntimeConstraintFiles) > 0,
		responseRules:      len(options.ResponseRuleFiles) > 0,
	})
	if err != nil {
		return PromptConfig{}, fmt.Errorf("parse prompts config %q: %w", resolvedPath, err)
	}
	if err := applyPromptOverrides(&cfg, options); err != nil {
		return PromptConfig{}, err
	}
	return cfg, nil
}

func applyPromptOverrides(cfg *PromptConfig, options PromptLoadOptions) error {
	overrides := []struct {
		files []string
		name  string
		set   func(string)
	}{
		{
			files: options.CoreFiles,
			name:  "core job",
			set:   func(content string) { cfg.System.CoreJob = content },
		},
		{
			files: options.RuntimeConstraintFiles,
			name:  "runtime constraints",
			set:   func(content string) { cfg.System.RuntimeConstraints = content },
		},
		{
			files: options.ResponseRuleFiles,
			name:  "response rules",
			set:   func(content string) { cfg.System.ResponseRules = content },
		},
	}

	for _, override := range overrides {
		if len(override.files) == 0 {
			continue
		}
		content, err := loadPromptSectionFromFiles(options.CoreDir, override.files)
		if err != nil {
			return fmt.Errorf("load %s: %w", override.name, err)
		}
		override.set(content)
	}
	return nil
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
	candidates := []string{primary, filepath.Join("core", "bridge", primary)}
	if sourceRoot, ok := bridgeSourceRoot(); ok {
		candidates = append(candidates, filepath.Join(sourceRoot, primary))
	}
	return uniquePathList(candidates)
}

func bridgeSourceRoot() (string, bool) {
	_, file, _, ok := goruntime.Caller(0)
	if !ok {
		return "", false
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..")), true
}

func uniquePathList(candidates []string) []string {
	seen := make(map[string]struct{}, len(candidates))
	out := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		cleaned := filepath.Clean(strings.TrimSpace(candidate))
		if cleaned == "" {
			continue
		}
		if _, ok := seen[cleaned]; ok {
			continue
		}
		seen[cleaned] = struct{}{}
		out = append(out, cleaned)
	}
	return out
}

func parsePromptYAML(raw []byte, optional promptSectionOptions) (PromptConfig, error) {
	cfg := PromptConfig{}
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return PromptConfig{}, err
	}
	if strings.TrimSpace(cfg.Version) == "" {
		cfg.Version = defaultPromptVersion
	}
	if strings.TrimSpace(cfg.System.Default) == "" {
		return PromptConfig{}, errSystemDefaultRequired
	}

	requirements := []promptSectionRequirement{
		{
			field:       "system.core_job",
			placeholder: "{{core_job}}",
			value:       cfg.System.CoreJob,
			optional:    optional.coreJob,
		},
		{
			field:       "system.runtime_constraints",
			placeholder: "{{runtime_constraints}}",
			value:       cfg.System.RuntimeConstraints,
			optional:    optional.runtimeConstraints,
		},
		{
			field:       "system.response_rules",
			placeholder: "{{response_rules}}",
			value:       cfg.System.ResponseRules,
			optional:    optional.responseRules,
		},
	}
	for _, requirement := range requirements {
		if err := validatePromptSection(cfg.System.Default, requirement); err != nil {
			return PromptConfig{}, err
		}
	}
	return cfg, nil
}

func validatePromptSection(template string, requirement promptSectionRequirement) error {
	if !strings.Contains(template, requirement.placeholder) {
		return nil
	}
	if requirement.optional || strings.TrimSpace(requirement.value) != "" {
		return nil
	}
	return fmt.Errorf(
		"%s is required when system.default references %s",
		requirement.field,
		requirement.placeholder,
	)
}

func loadPromptSectionFromFiles(baseDir string, files []string) (string, error) {
	parts := make([]string, 0, len(files))
	root := strings.TrimSpace(baseDir)
	if root != "" {
		root = filepath.Clean(root)
	}
	for _, file := range files {
		path := strings.TrimSpace(file)
		if path == "" {
			return "", errors.New("prompt section file name is empty")
		}
		if !filepath.IsAbs(path) {
			if root == "" {
				return "", errors.New("prompts dir is empty for relative prompt section file")
			}
			path = filepath.Join(root, path)
		}
		cleaned := filepath.Clean(path)
		raw, err := os.ReadFile(cleaned)
		if err != nil {
			return "", fmt.Errorf("read prompt section file %s: %w", cleaned, err)
		}
		content := strings.TrimSpace(string(raw))
		if content == "" {
			return "", fmt.Errorf("prompt section file %s is empty", cleaned)
		}
		parts = append(parts, content)
	}
	if len(parts) == 0 {
		return "", errors.New("no prompt section files provided")
	}
	return strings.Join(parts, "\n\n"), nil
}
