package sessions

import "strings"

func addSessionSourceID(ids map[string]struct{}, value string) {
	id := strings.TrimSpace(value)
	if id == "" {
		return
	}
	ids[id] = struct{}{}
}

func stringFromRecord(record map[string]any, key string) string {
	return stringFromAny(record[key])
}

func stringFromAny(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func recordFromAny(value any) map[string]any {
	record, ok := value.(map[string]any)
	if ok {
		return record
	}
	return nil
}

func arrayFromAny(value any) []any {
	items, ok := value.([]any)
	if ok {
		return items
	}
	return nil
}

func mapKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}
