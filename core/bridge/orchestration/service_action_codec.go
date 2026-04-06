// Request/response codec utilities for validating and normalizing bus envelopes.

package orchestration

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// decodeActionParams 将 raw params 解码为用例参数类型。
func decodeActionParams[T any](raw json.RawMessage) (T, error) {
	var params T
	if err := decodeParams(raw, &params); err != nil {
		return params, err
	}
	return params, nil
}

// validateBusRequest 做 envelope 级别校验，要求 action/trace_id 存在且 params 为对象。
func validateBusRequest(req apiRequest) error {
	if strings.TrimSpace(req.Action) == "" {
		return errors.New("action is required")
	}
	if strings.TrimSpace(req.TraceID) == "" {
		return errors.New("trace_id is required")
	}

	source := bytes.TrimSpace(req.Params)
	if len(source) == 0 {
		return errors.New("params is required")
	}
	if bytes.Equal(source, []byte("null")) {
		return errors.New("params must be an object")
	}

	var object map[string]any
	if err := json.Unmarshal(source, &object); err != nil {
		return fmt.Errorf("params must be an object: %w", err)
	}
	if object == nil {
		return errors.New("params must be an object")
	}
	return nil
}

// decodeParams 解码 params 并禁止未知字段；空/null 统一按空对象处理。
func decodeParams(raw json.RawMessage, target any) error {
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
