package workflow

import (
	"encoding/json"
	"fmt"
	"regexp"

	bridgeTasks "ghost-os/bridge/tasks"
)

var inputNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func validateStartNode(node Node) error {
	if node.Start == nil {
		return nil
	}
	seen := make(map[string]bool, len(node.Start.Inputs))
	for index, input := range node.Start.Inputs {
		if err := validateInputVariable(input, seen); err != nil {
			return fmt.Errorf("%w: workflow start node %q input[%d] %v", bridgeTasks.ErrInvalidTaskConfig, node.ID, index, err)
		}
	}
	return nil
}

func validateInputVariable(input InputVariable, seen map[string]bool) error {
	if !inputNamePattern.MatchString(input.Name) {
		return fmt.Errorf("name %q must match ^[A-Za-z_][A-Za-z0-9_]*$", input.Name)
	}
	if seen[input.Name] {
		return fmt.Errorf("duplicate name %q", input.Name)
	}
	seen[input.Name] = true
	if !isInputTypeSupported(input.Type) {
		return fmt.Errorf("name %q uses unsupported type %q", input.Name, input.Type)
	}
	if len(input.Default) == 0 {
		return nil
	}
	if err := validateInputDefault(input.Type, input.Default); err != nil {
		return fmt.Errorf("name %q default %v", input.Name, err)
	}
	return nil
}

func isInputTypeSupported(inputType string) bool {
	switch inputType {
	case InputTypeString, InputTypeNumber, InputTypeBoolean, InputTypeObject, InputTypeArray:
		return true
	default:
		return false
	}
}

func validateInputDefault(inputType string, raw json.RawMessage) error {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return fmt.Errorf("must be valid JSON")
	}
	if inputDefaultMatchesType(inputType, value) {
		return nil
	}
	return fmt.Errorf("must match declared type %q", inputType)
}

func inputDefaultMatchesType(inputType string, value any) bool {
	switch inputType {
	case InputTypeString:
		_, ok := value.(string)
		return ok
	case InputTypeNumber:
		_, ok := value.(float64)
		return ok
	case InputTypeBoolean:
		_, ok := value.(bool)
		return ok
	case InputTypeObject:
		_, ok := value.(map[string]any)
		return ok
	case InputTypeArray:
		_, ok := value.([]any)
		return ok
	default:
		return false
	}
}
