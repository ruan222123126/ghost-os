package orchestration

import "strings"

func normalizeTaskRuntimeOverrides(input *TaskRuntimeOverrides) (*TaskRuntimeOverrides, error) {
	if input == nil {
		return nil, nil
	}
	allowlist, _, err := normalizeConfiguredToolLists(input.ToolAllowlist, nil)
	if err != nil {
		return nil, err
	}
	model := strings.TrimSpace(input.Model)
	if model == "" && len(allowlist) == 0 {
		return nil, nil
	}
	return &TaskRuntimeOverrides{
		Model:         model,
		ToolAllowlist: allowlist,
	}, nil
}
