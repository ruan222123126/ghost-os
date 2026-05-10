package workflow

import (
	"regexp"
	"strings"
)

var findIconTemplatePattern = regexp.MustCompile(`\$\{([^{}]+)\}`)

func ResolveFindIconTemplateValue(value any, findIcon any) (any, error) {
	switch typed := value.(type) {
	case string:
		return ResolveFindIconTemplateString(typed, findIcon)
	case map[string]any:
		return resolveFindIconTemplateMap(typed, findIcon)
	case []any:
		return resolveFindIconTemplateArray(typed, findIcon)
	default:
		return value, nil
	}
}

func resolveFindIconTemplateMap(input map[string]any, findIcon any) (map[string]any, error) {
	output := make(map[string]any, len(input))
	for key, value := range input {
		resolvedValue, err := ResolveFindIconTemplateValue(value, findIcon)
		if err != nil {
			return nil, err
		}
		output[key] = resolvedValue
	}
	return output, nil
}

func resolveFindIconTemplateArray(input []any, findIcon any) ([]any, error) {
	output := make([]any, len(input))
	for index, value := range input {
		resolvedValue, err := ResolveFindIconTemplateValue(value, findIcon)
		if err != nil {
			return nil, err
		}
		output[index] = resolvedValue
	}
	return output, nil
}

func ResolveFindIconTemplateString(input string, findIcon any) (any, error) {
	matches := findIconTemplatePattern.FindAllStringSubmatchIndex(input, -1)
	if len(matches) == 0 {
		return input, nil
	}
	if isWholeTemplate(input, matches) {
		return resolveFindIconTemplatePathOrLiteral(input[matches[0][2]:matches[0][3]], input, findIcon)
	}
	return resolveInterpolatedFindIconTemplate(input, matches, findIcon)
}

func isWholeTemplate(input string, matches [][]int) bool {
	return matches[0][0] == 0 && matches[0][1] == len(input) && len(matches) == 1
}

func resolveFindIconTemplatePathOrLiteral(path string, literal string, findIcon any) (any, error) {
	resolved, changed, err := ResolveFindIconPath(path, findIcon)
	if err != nil {
		return nil, err
	}
	if !changed {
		return literal, nil
	}
	return resolved, nil
}

func resolveInterpolatedFindIconTemplate(input string, matches [][]int, findIcon any) (string, error) {
	var builder strings.Builder
	lastIndex := 0
	for _, match := range matches {
		builder.WriteString(input[lastIndex:match[0]])
		if err := writeFindIconTemplateMatch(&builder, input, match, findIcon); err != nil {
			return "", err
		}
		lastIndex = match[1]
	}
	builder.WriteString(input[lastIndex:])
	return builder.String(), nil
}

func writeFindIconTemplateMatch(builder *strings.Builder, input string, match []int, findIcon any) error {
	resolved, changed, err := ResolveFindIconPath(input[match[2]:match[3]], findIcon)
	if err != nil {
		return err
	}
	if changed {
		builder.WriteString(StringifyFindIconTemplateValue(resolved))
		return nil
	}
	builder.WriteString(input[match[0]:match[1]])
	return nil
}
