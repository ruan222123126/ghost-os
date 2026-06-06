package runtimeopts

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"ghost-os/bridge/taskdefs"
)

func TestNormalizeTaskOverridesTrimsSortsAndDropsEmptyValues(t *testing.T) {
	allowlistOnly := true
	maxTurns := 3
	normalized, err := NormalizeTaskOverrides(&taskdefs.TaskRuntimeOverrides{
		ProviderName:      " openai-main ",
		Model:             " gpt-5 ",
		SystemPrompt:      " be direct ",
		PresetID:          " preset-1 ",
		ToolAllowlist:     []string{" script_exec ", "", "web_search", "script_exec"},
		ToolAllowlistOnly: &allowlistOnly,
		MaxTurns:          &maxTurns,
	}, OverrideNormalizationOptions{
		ValidToolNames: []string{"web_search", "script_exec"},
	})
	if err != nil {
		t.Fatalf("normalize overrides: %v", err)
	}
	if normalized.ProviderName != "openai-main" || normalized.Model != "gpt-5" {
		t.Fatalf("unexpected provider/model: %#v", normalized)
	}
	if normalized.SystemPrompt != "be direct" || normalized.PresetID != "preset-1" {
		t.Fatalf("unexpected prompt/preset: %#v", normalized)
	}
	if !reflect.DeepEqual(normalized.ToolAllowlist, []string{"script_exec", "web_search"}) {
		t.Fatalf("unexpected allowlist: %#v", normalized.ToolAllowlist)
	}
	if normalized.ToolAllowlistOnly == nil || !*normalized.ToolAllowlistOnly {
		t.Fatalf("expected strict tool allowlist, got %#v", normalized.ToolAllowlistOnly)
	}
	if normalized.MaxTurns == nil || *normalized.MaxTurns != 3 {
		t.Fatalf("unexpected max_turns: %#v", normalized.MaxTurns)
	}
}

func TestNormalizeTaskOverridesValidatesProviderModelPair(t *testing.T) {
	_, err := NormalizeTaskOverrides(&taskdefs.TaskRuntimeOverrides{
		Model: "gpt-5",
	}, OverrideNormalizationOptions{RequireProviderModelPair: true})
	if err == nil || err.Error() != "provider_name and model must be set together" {
		t.Fatalf("expected provider/model pair error, got %v", err)
	}
}

func TestNormalizeTaskOverridesDropsMaxTurnsForOrchestrationMembers(t *testing.T) {
	maxTurns := 9
	normalized, err := NormalizeTaskOverrides(&taskdefs.TaskRuntimeOverrides{
		MaxTurns: &maxTurns,
	}, OverrideNormalizationOptions{DropMaxTurns: true})
	if err != nil {
		t.Fatalf("normalize overrides: %v", err)
	}
	if normalized != nil {
		t.Fatalf("expected empty overrides after dropping max_turns, got %#v", normalized)
	}
}

func TestValidateOverridesAgainstCatalogUsesActiveProviderForModelOnly(t *testing.T) {
	err := ValidateOverridesAgainstCatalog(&taskdefs.TaskRuntimeOverrides{Model: "gpt-5"}, ProviderCatalog{
		ActiveProvider: "openai-main",
		Providers: []ProviderRecord{{
			Name:   "openai-main",
			Models: []string{"gpt-5"},
		}},
	}, false)
	if err != nil {
		t.Fatalf("validate overrides: %v", err)
	}
}

func TestNormalizeRequestOptionsValidatesProjectRoot(t *testing.T) {
	dir := t.TempDir()
	options, err := NormalizeRequestOptions(filepath.Join(dir, "."))
	if err != nil {
		t.Fatalf("normalize request options: %v", err)
	}
	if options.ProjectRoot != filepath.Clean(dir) {
		t.Fatalf("unexpected project_root: got %q want %q", options.ProjectRoot, filepath.Clean(dir))
	}

	_, err = NormalizeRequestOptions("relative")
	if !errors.Is(err, ErrProjectRootAbsolute) {
		t.Fatalf("expected absolute path error, got %v", err)
	}
}

func TestNormalizeRetryPolicyRejectsNegativeValues(t *testing.T) {
	if _, err := NormalizeRetryPolicy(-1, 0); err == nil {
		t.Fatalf("expected retry count error")
	}
	if _, err := NormalizeRetryPolicy(0, -1); err == nil {
		t.Fatalf("expected retry interval error")
	}
}
