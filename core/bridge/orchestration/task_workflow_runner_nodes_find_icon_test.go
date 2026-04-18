package orchestration

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPrepareWorkflowToolArgumentsUploadsFindIconTemplate(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	input := map[string]any{
		"mode":   "atomic",
		"action": "find_icon",
		"params": map[string]any{
			"workflow_template_data_url": "data:image/png;base64,R2hvc3Q=",
			"template_filename":          "icon.png",
			"template_mime_type":         "image/png",
			"threshold":                  0.91,
		},
	}
	original := cloneTaskActionParams(input)

	prepared, err := prepareWorkflowToolArguments(screenControlToolID, input)
	if err != nil {
		t.Fatalf("prepareWorkflowToolArguments returned error: %v", err)
	}

	params := decodeWorkflowToolParams(t, prepared)
	templatePath := strings.TrimSpace(workflowMapString(params, "template_path"))
	if templatePath == "" {
		t.Fatalf("expected template_path to be generated, got: %+v", params)
	}
	if !strings.HasPrefix(templatePath, filepath.Join(homeDir, ".ghost-os")) {
		t.Fatalf("unexpected template path: %q", templatePath)
	}
	if _, err := os.Stat(templatePath); err != nil {
		t.Fatalf("template path should exist: %v", err)
	}
	if _, exists := params["workflow_template_data_url"]; exists {
		t.Fatalf("workflow_template_data_url should be removed after upload: %+v", params)
	}
	if params["threshold"] != 0.91 {
		t.Fatalf("expected threshold to be preserved, got: %+v", params)
	}
	if !reflect.DeepEqual(input, original) {
		t.Fatalf("input arguments should remain immutable: before=%+v after=%+v", original, input)
	}
}

func TestPrepareWorkflowToolArgumentsRejectsInvalidFindIconUploadData(t *testing.T) {
	_, err := prepareWorkflowToolArguments(screenControlToolID, map[string]any{
		"mode":   "atomic",
		"action": "find_icon",
		"params": map[string]any{
			"workflow_template_data_url": "invalid-data-url",
			"template_mime_type":         "image/png",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "data_url must start with data:") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrepareWorkflowToolArgumentsRejectsLegacyFindIconUploadKeys(t *testing.T) {
	_, err := prepareWorkflowToolArguments(screenControlToolID, map[string]any{
		"mode":   "atomic",
		"action": "find_icon",
		"params": map[string]any{
			"template_data_url": "data:image/png;base64,R2hvc3Q=",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "legacy find_icon data_url keys are not supported") {
		t.Fatalf("unexpected error for template_data_url: %v", err)
	}

	_, err = prepareWorkflowToolArguments(screenControlToolID, map[string]any{
		"mode":   "atomic",
		"action": "find_icon",
		"params": map[string]any{
			"data_url": "data:image/png;base64,R2hvc3Q=",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "legacy find_icon data_url keys are not supported") {
		t.Fatalf("unexpected error for data_url: %v", err)
	}
}

func TestPrepareWorkflowToolArgumentsSkipsNonScreenControlTools(t *testing.T) {
	input := map[string]any{
		"command": "pwd",
	}
	prepared, err := prepareWorkflowToolArguments("script_exec", input)
	if err != nil {
		t.Fatalf("prepareWorkflowToolArguments returned error: %v", err)
	}
	if !reflect.DeepEqual(prepared, input) {
		t.Fatalf("unexpected prepared arguments: got=%+v want=%+v", prepared, input)
	}
}

func decodeWorkflowToolParams(t *testing.T, arguments map[string]any) map[string]any {
	t.Helper()
	params, ok := arguments["params"].(map[string]any)
	if !ok {
		t.Fatalf("params should be an object, got: %T", arguments["params"])
	}
	return params
}
