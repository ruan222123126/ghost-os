package tools

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

func decodeTaskManageArgs(raw json.RawMessage, target *taskManageArgs) error {
	source := bytes.TrimSpace(raw)
	if len(source) == 0 || bytes.Equal(source, []byte("null")) {
		source = []byte("{}")
	}
	decoder := json.NewDecoder(bytes.NewReader(source))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err == nil {
		return errors.New("invalid params: multiple JSON values are not allowed")
	} else if !errors.Is(err, io.EOF) {
		return fmt.Errorf("invalid params: %w", err)
	}
	return nil
}

func validateTaskManageSchedule(intervalSeconds *int, cronExpr *string, requireOne bool) error {
	hasInterval := intervalSeconds != nil
	hasCron := cronExpr != nil
	if hasInterval && *intervalSeconds < 1 {
		return fmt.Errorf("interval_seconds must be >= 1")
	}
	if hasCron && strings.TrimSpace(*cronExpr) == "" {
		return fmt.Errorf("cron_expr must not be empty")
	}
	if hasInterval && hasCron {
		return fmt.Errorf("exactly one of interval_seconds or cron_expr is allowed")
	}
	if requireOne && !hasInterval && !hasCron {
		return fmt.Errorf("exactly one of interval_seconds or cron_expr is required")
	}
	return nil
}

func ensureAgentTaskPayload(task TaskPayload) error {
	if normalizeTaskManageKind(task.TaskKind) != taskManageKindAgentMessage {
		return fmt.Errorf("task_manage only supports agent_message tasks")
	}
	return nil
}

func normalizeTaskManageKind(kind string) string {
	if strings.TrimSpace(kind) == "" {
		return taskManageKindAgentMessage
	}
	return strings.TrimSpace(kind)
}

func marshalTaskManageOutput(payload any) (string, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode task result: %w", err)
	}
	return string(encoded), nil
}

func trimmedOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func optionalIntValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
