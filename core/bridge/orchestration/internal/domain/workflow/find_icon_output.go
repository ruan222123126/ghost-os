package workflow

import "fmt"

type FindIconMatch struct {
	X         float64
	Y         float64
	DisplayID int
}

func ExtractFindIconOutput(node Node, screenControlToolID string, outputValue any) (any, bool) {
	if node.Type != NodeTypeTool || node.Tool == nil || node.Tool.ToolName != screenControlToolID {
		return nil, false
	}
	return extractFindIconOutputValue(outputValue)
}

func extractFindIconOutputValue(outputValue any) (any, bool) {
	record, ok := outputValue.(map[string]any)
	if !ok {
		return nil, false
	}
	if _, exists := record["matches"]; exists {
		return buildFindIconReferenceValue(record)
	}
	if value, ok := extractFindIconOutputFromTypedSteps(record); ok {
		return value, true
	}
	return extractFindIconOutputFromAnySteps(record)
}

func extractFindIconOutputFromTypedSteps(record map[string]any) (any, bool) {
	rawSteps, ok := record["steps"].([]map[string]any)
	if !ok {
		return nil, false
	}
	for index := len(rawSteps) - 1; index >= 0; index-- {
		if rawSteps[index]["action"] == "find_icon" {
			return buildFindIconReferenceValue(rawSteps[index]["output"])
		}
	}
	return nil, false
}

func extractFindIconOutputFromAnySteps(record map[string]any) (any, bool) {
	steps, ok := record["steps"].([]any)
	if !ok {
		return nil, false
	}
	for index := len(steps) - 1; index >= 0; index-- {
		step, ok := steps[index].(map[string]any)
		if ok && step["action"] == "find_icon" {
			return buildFindIconReferenceValue(step["output"])
		}
	}
	return nil, false
}

func buildFindIconReferenceValue(outputValue any) (map[string]any, bool) {
	match, err := FindIconMatchCenter(outputValue)
	if err != nil {
		return nil, false
	}
	value := map[string]any{"x": match.X, "y": match.Y}
	if match.DisplayID >= 0 {
		value["display_id"] = match.DisplayID
	}
	return value, true
}

func FindIconMatchCenter(output any) (FindIconMatch, error) {
	record, ok := output.(map[string]any)
	if !ok {
		return FindIconMatch{}, fmt.Errorf("find_icon output must be an object")
	}
	return findIconMatchCenterFromRecord(record)
}

func findIconMatchCenterFromRecord(record map[string]any) (FindIconMatch, error) {
	first, err := firstFindIconMatch(record)
	if err != nil {
		return FindIconMatch{}, err
	}
	center, ok := first["center"].(map[string]any)
	if !ok {
		return FindIconMatch{}, fmt.Errorf("find_icon output field matches[0].center must be an object")
	}
	return decodeFindIconCenter(record, center)
}

func firstFindIconMatch(record map[string]any) (map[string]any, error) {
	matches, ok := record["matches"].([]any)
	if !ok || len(matches) == 0 {
		return nil, fmt.Errorf("find_icon output field matches is empty")
	}
	first, ok := matches[0].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("find_icon output field matches[0] must be an object")
	}
	return first, nil
}

func decodeFindIconCenter(record map[string]any, center map[string]any) (FindIconMatch, error) {
	x, ok := AnyNumber(center["x"])
	if !ok {
		return FindIconMatch{}, fmt.Errorf("find_icon output field matches[0].center.x must be a number")
	}
	y, ok := AnyNumber(center["y"])
	if !ok {
		return FindIconMatch{}, fmt.Errorf("find_icon output field matches[0].center.y must be a number")
	}
	displayID := -1
	if value, ok := AnyInteger(record["display_id"]); ok {
		displayID = value
	}
	return FindIconMatch{X: x, Y: y, DisplayID: displayID}, nil
}
