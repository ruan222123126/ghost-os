package taskdefs

func CloneActionParams(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = CloneJSONValue(value)
	}
	return out
}

func CloneJSONValue(input any) any {
	switch typed := input.(type) {
	case map[string]any:
		return CloneActionParams(typed)
	case []any:
		out := make([]any, len(typed))
		for i := range typed {
			out[i] = CloneJSONValue(typed[i])
		}
		return out
	default:
		return typed
	}
}
