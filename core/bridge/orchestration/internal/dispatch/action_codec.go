package dispatch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"ghost-os/bridge/orchestration/internal/contracts/bus"
)

func DecodeActionParams[T any](raw json.RawMessage) (T, error) {
	var params T
	if err := DecodeParams(raw, &params); err != nil {
		return params, err
	}
	return params, nil
}

func ValidateBusRequest(req bus.RequestEnvelope) error {
	if strings.TrimSpace(req.Action) == "" {
		return errors.New("action is required")
	}
	if strings.TrimSpace(req.TraceID) == "" {
		return errors.New("trace_id is required")
	}
	return validateParamsObject(req.Params)
}

func DecodeParams(raw json.RawMessage, target any) error {
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

func validateParamsObject(raw json.RawMessage) error {
	source := bytes.TrimSpace(raw)
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
