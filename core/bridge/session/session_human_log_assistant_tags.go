package session

import "strings"

func extractAssistantVisibleTextAndTagCalls(raw string) (string, []assistantTagCall) {
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}

	var (
		visible strings.Builder
		calls   []assistantTagCall
		cursor  int
	)
	for cursor < len(raw) {
		openIndex := strings.Index(raw[cursor:], "<t:")
		if openIndex < 0 {
			visible.WriteString(raw[cursor:])
			break
		}
		openIndex += cursor
		visible.WriteString(raw[cursor:openIndex])

		idStart := openIndex + len("<t:")
		idEnd := strings.Index(raw[idStart:], ">")
		if idEnd < 0 {
			visible.WriteString(raw[openIndex:])
			break
		}
		idEnd += idStart

		closeIndex := strings.Index(raw[idEnd+1:], "</t>")
		if closeIndex < 0 {
			visible.WriteString(raw[openIndex:])
			break
		}
		closeIndex += idEnd + 1

		calls = append(calls, assistantTagCall{
			ID:        strings.TrimSpace(raw[idStart:idEnd]),
			Arguments: strings.TrimSpace(raw[idEnd+1 : closeIndex]),
		})
		cursor = closeIndex + len("</t>")
	}
	return strings.TrimSpace(visible.String()), calls
}
