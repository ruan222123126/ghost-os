package orchestration

import (
	"encoding/json"
	"fmt"
	"regexp"
)

var workflowInputNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func validateWorkflowStartNode(node WorkflowNode) error {
	if node.Start == nil {
		return nil
	}
	seen := make(map[string]bool, len(node.Start.Inputs))
	for index, input := range node.Start.Inputs {
		if err := validateWorkflowInputVariable(input, seen); err != nil {
			return fmt.Errorf("%w: workflow start node %q input[%d] %v", ErrInvalidTaskConfig, node.ID, index, err)
		}
	}
	return nil
}

func validateWorkflowInputVariable(input WorkflowInputVariable, seen map[string]bool) error {
	if !workflowInputNamePattern.MatchString(input.Name) {
		return fmt.Errorf("name %q must match ^[A-Za-z_][A-Za-z0-9_]*$", input.Name)
	}
	if seen[input.Name] {
		return fmt.Errorf("duplicate name %q", input.Name)
	}
	seen[input.Name] = true
	if !isWorkflowInputTypeSupported(input.Type) {
		return fmt.Errorf("name %q uses unsupported type %q", input.Name, input.Type)
	}
	if len(input.Default) == 0 {
		return nil
	}
	if err := validateWorkflowInputDefault(input.Type, input.Default); err != nil {
		return fmt.Errorf("name %q default %v", input.Name, err)
	}
	return nil
}

func isWorkflowInputTypeSupported(inputType string) bool {
	switch inputType {
	case workflowInputTypeString, workflowInputTypeNumber, workflowInputTypeBoolean, workflowInputTypeObject, workflowInputTypeArray:
		return true
	default:
		return false
	}
}

func validateWorkflowInputDefault(inputType string, raw json.RawMessage) error {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return fmt.Errorf("must be valid JSON")
	}
	if workflowInputDefaultMatchesType(inputType, value) {
		return nil
	}
	return fmt.Errorf("must match declared type %q", inputType)
}

func workflowInputDefaultMatchesType(inputType string, value any) bool {
	switch inputType {
	case workflowInputTypeString:
		_, ok := value.(string)
		return ok
	case workflowInputTypeNumber:
		_, ok := value.(float64)
		return ok
	case workflowInputTypeBoolean:
		_, ok := value.(bool)
		return ok
	case workflowInputTypeObject:
		_, ok := value.(map[string]any)
		return ok
	case workflowInputTypeArray:
		_, ok := value.([]any)
		return ok
	default:
		return false
	}
}
