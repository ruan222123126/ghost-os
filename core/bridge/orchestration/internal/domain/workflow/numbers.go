package workflow

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
