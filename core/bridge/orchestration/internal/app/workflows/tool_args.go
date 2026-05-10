package workflows

import (
	"encoding/json"
	"fmt"
	"strings"

	bridgeTasks "ghost-os/bridge/tasks"
)

func EncodeToolArguments(arguments map[string]any) (json.RawMessage, error) {
	if len(arguments) == 0 {
		return json.RawMessage(`{}`), nil
	}
	encoded, err := json.Marshal(arguments)
	if err != nil {
		return nil, fmt.Errorf("encode workflow tool arguments: %w", err)
	}
	return encoded, nil
}

func PrepareToolArguments(
	toolName string,
	arguments map[string]any,
	uploader TemplateUploader,
) (map[string]any, error) {
	cloned := bridgeTasks.CloneActionParams(arguments)
	if len(cloned) == 0 {
		return map[string]any{}, nil
	}
	if strings.TrimSpace(toolName) != ScreenControlToolID {
		return cloned, nil
	}
	return prepareScreenControlFindIconArguments(cloned, uploader)
}

func prepareScreenControlFindIconArguments(
	arguments map[string]any,
	uploader TemplateUploader,
) (map[string]any, error) {
	if !isScreenFindIconAction(arguments) {
		return arguments, nil
	}
	rawParams, ok := arguments[ScreenControlParamsKey].(map[string]any)
	if !ok {
		return arguments, nil
	}
	params, changed, err := ensureFindIconTemplatePath(rawParams, uploader)
	if err != nil {
		return nil, err
	}
	if !changed {
		return arguments, nil
	}
	arguments[ScreenControlParamsKey] = params
	return arguments, nil
}

func isScreenFindIconAction(arguments map[string]any) bool {
	action := strings.ToLower(mapString(arguments, ScreenControlActionKey))
	if action != "find_icon" && action != "click_icon" {
		return false
	}
	mode := strings.ToLower(mapString(arguments, "mode"))
	return mode == "" || mode == ScreenControlAtomicMode
}
