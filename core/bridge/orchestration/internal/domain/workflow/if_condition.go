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

func AnyNumber(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case int32:
		return float64(typed), true
	default:
		return 0, false
	}
}

func AnyInteger(value any) (int, bool) {
	number, ok := AnyNumber(value)
	if !ok {
		return 0, false
	}
	return int(number), true
}
