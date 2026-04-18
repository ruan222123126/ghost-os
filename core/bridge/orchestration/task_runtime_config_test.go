package orchestration

import (
	"strings"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
)

func TestValidateWorkflowTaskRuntimeAllowsAllToolsWhenAllowlistEmpty(t *testing.T) {
	definition := workflowWithToolNode("rss_fetch")

	err := validateWorkflowTaskRuntime(definition, bridgeconfig.TaskConfig{})
	if err != nil {
		t.Fatalf("expected empty allowlist to allow all tools, got %v", err)
	}
}

func TestValidateWorkflowTaskRuntimeRejectsToolOutsideExplicitAllowlist(t *testing.T) {
	definition := workflowWithToolNode("rss_fetch")
	cfg := bridgeconfig.TaskConfig{WorkflowToolAllowlist: []string{"script_exec"}}

	err := validateWorkflowTaskRuntime(definition, cfg)
	if err == nil {
		t.Fatal("expected workflow tool validation to fail")
	}
	if !strings.Contains(err.Error(), `workflow tool "rss_fetch" is not allowed`) {
		t.Fatalf("unexpected error: %v", err)
	}
}
