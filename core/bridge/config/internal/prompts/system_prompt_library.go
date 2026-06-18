package prompts

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const (
	maxActivePromptPerInsertPoint = 1
	coreJobPromptLibraryCardID    = "core-job"
	coreJobPromptLibraryCardName  = "Core Job"
)

func parsePromptLibrary(raw string) ([]SystemPromptLibraryItem, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return []SystemPromptLibraryItem{}, nil
	}

	decoder := json.NewDecoder(strings.NewReader(trimmed))
	decoder.DisallowUnknownFields()
	var library []SystemPromptLibraryItem
	if err := decoder.Decode(&library); err != nil {
		return nil, fmt.Errorf("%w: parse prompt_library: %v", errSystemPromptLibraryInvalid, err)
	}
	if err := rejectTrailingPromptLibraryJSON(decoder); err != nil {
		return nil, err
	}
	return library, nil
}

func rejectTrailingPromptLibraryJSON(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err == io.EOF {
		return nil
	} else if err != nil {
		return fmt.Errorf("%w: parse prompt_library: %v", errSystemPromptLibraryInvalid, err)
	}
	return fmt.Errorf("%w: prompt_library must be a single JSON array", errSystemPromptLibraryInvalid)
}

func marshalPromptLibrary(library []SystemPromptLibraryItem) (string, error) {
	payload, err := json.Marshal(library)
	if err != nil {
		return "", fmt.Errorf("%w: marshal prompt_library: %v", errSystemPromptLibraryInvalid, err)
	}
	return string(payload), nil
}

func normalizePromptLibrary(library []SystemPromptLibraryItem) ([]SystemPromptLibraryItem, error) {
	normalized := make([]SystemPromptLibraryItem, 0, len(library))
	ids := make(map[string]struct{}, len(library))
	activeCounts := make(map[SystemPromptInsertPoint]int)

	for index, item := range library {
		next, err := normalizePromptLibraryItem(item, index)
		if err != nil {
			return nil, err
		}
		if _, exists := ids[next.ID]; exists {
			return nil, newPromptLibraryValidationError("item %d id %q is duplicated", index, next.ID)
		}
		ids[next.ID] = struct{}{}
		if next.Active {
			activeCounts[next.InsertPoint] += 1
			if !insertPointAllowsMultipleActive(next.InsertPoint) &&
				activeCounts[next.InsertPoint] > maxActivePromptPerInsertPoint {
				return nil, newPromptLibraryValidationError(
					"insert_point %q has more than one active item",
					next.InsertPoint,
				)
			}
		}
		normalized = append(normalized, next)
	}
	return normalized, nil
}

func NormalizePromptLibrary(library []SystemPromptLibraryItem) ([]SystemPromptLibraryItem, error) {
	return normalizePromptLibrary(library)
}

func normalizePromptLibraryItem(item SystemPromptLibraryItem, index int) (SystemPromptLibraryItem, error) {
	id := strings.TrimSpace(item.ID)
	if id == "" {
		return SystemPromptLibraryItem{}, newPromptLibraryValidationError("item %d id is required", index)
	}

	name := strings.TrimSpace(item.Name)
	if name == "" {
		return SystemPromptLibraryItem{}, newPromptLibraryValidationError("item %d name is required", index)
	}

	insertPoint := SystemPromptInsertPoint(strings.TrimSpace(string(item.InsertPoint)))
	if !isAllowedPromptInsertPoint(insertPoint) {
		return SystemPromptLibraryItem{}, newPromptLibraryValidationError(
			"item %d insert_point %q is invalid",
			index,
			insertPoint,
		)
	}

	return SystemPromptLibraryItem{
		ID:          id,
		Name:        name,
		InsertPoint: insertPoint,
		Content:     strings.TrimSpace(item.Content),
		Active:      item.Active,
	}, nil
}

func isAllowedPromptInsertPoint(insertPoint SystemPromptInsertPoint) bool {
	return insertPoint == SystemPromptInsertPointRule ||
		insertPoint == SystemPromptInsertPointCoreJob ||
		insertPoint == SystemPromptInsertPointMemory ||
		insertPoint == SystemPromptInsertPointContext
}

func insertPointAllowsMultipleActive(insertPoint SystemPromptInsertPoint) bool {
	return insertPoint == SystemPromptInsertPointContext
}

func buildCoreJobPromptLibrary(rawCorePrompt *string) []SystemPromptLibraryItem {
	return []SystemPromptLibraryItem{
		{
			ID:          coreJobPromptLibraryCardID,
			Name:        coreJobPromptLibraryCardName,
			InsertPoint: SystemPromptInsertPointCoreJob,
			Content:     trimSystemPromptValue(rawCorePrompt),
			Active:      true,
		},
	}
}

func compileCorePromptFromLibrary(library []SystemPromptLibraryItem) string {
	for _, item := range library {
		if item.InsertPoint == SystemPromptInsertPointCoreJob && item.Active {
			return strings.TrimSpace(item.Content)
		}
	}
	return ""
}

func CompileCorePromptFromLibrary(library []SystemPromptLibraryItem) string {
	return compileCorePromptFromLibrary(library)
}

func newPromptLibraryValidationError(format string, args ...any) error {
	return fmt.Errorf("%w: %s", errSystemPromptLibraryInvalid, fmt.Sprintf(format, args...))
}
