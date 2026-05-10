package workflow

import (
	"fmt"
	"strings"
)

func EvaluateIfCondition(operator string, source string, value string) (bool, error) {
	normalizedOperator := strings.TrimSpace(operator)
	sourceText := strings.TrimSpace(source)
	targetValue := strings.TrimSpace(value)
	switch normalizedOperator {
	case IfOperatorEquals:
		return sourceText == targetValue, nil
	case IfOperatorNotEquals:
		return sourceText != targetValue, nil
	case IfOperatorContains:
		return strings.Contains(sourceText, targetValue), nil
	case IfOperatorNotContains:
		return !strings.Contains(sourceText, targetValue), nil
	case IfOperatorIsEmpty:
		return sourceText == "", nil
	case IfOperatorNotEmpty:
		return sourceText != "", nil
	default:
		return false, fmt.Errorf("unsupported operator %q", operator)
	}
}
