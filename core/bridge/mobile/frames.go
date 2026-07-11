package mobile

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const MaxFrameBytes = 1 << 20

const (
	FrameAuthChallenge = "auth_challenge"
	FrameAuthResponse  = "auth_response"
	FrameAuthOK        = "auth_ok"
	FrameAuthError     = "auth_error"
	FrameRequest       = "request"
	FrameResponse      = "response"
	FrameStreamStart   = "stream_start"
	FrameStreamEvent   = "stream_event"
	FrameStreamEnd     = "stream_end"
	FrameCancel        = "cancel"
)

type Frame struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	DeviceID  string          `json:"device_id,omitempty"`
	Challenge string          `json:"challenge,omitempty"`
	Signature string          `json:"signature,omitempty"`
	Action    string          `json:"action,omitempty"`
	Params    json.RawMessage `json:"params,omitempty"`
	TraceID   string          `json:"trace_id,omitempty"`
	Status    string          `json:"status,omitempty"`
	Payload   any             `json:"payload,omitempty"`
	Error     string          `json:"error,omitempty"`
	Event     string          `json:"event,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
}

type RequestPayload struct {
	Action  string          `json:"action"`
	Params  json.RawMessage `json:"params"`
	TraceID string          `json:"trace_id"`
}

func DecodeFrame(data []byte) (Frame, error) {
	if len(data) == 0 {
		return Frame{}, errors.New("frame is empty")
	}
	if len(data) > MaxFrameBytes {
		return Frame{}, fmt.Errorf("frame exceeds %d byte limit", MaxFrameBytes)
	}
	var frame Frame
	if err := json.Unmarshal(data, &frame); err != nil {
		return Frame{}, fmt.Errorf("decode frame: %w", err)
	}
	if err := ValidateFrame(frame); err != nil {
		return Frame{}, err
	}
	return frame, nil
}

func EncodeFrame(frame Frame) ([]byte, error) {
	if err := ValidateFrame(frame); err != nil {
		return nil, err
	}
	data, err := json.Marshal(frame)
	if err != nil {
		return nil, fmt.Errorf("encode frame: %w", err)
	}
	if len(data) > MaxFrameBytes {
		return nil, fmt.Errorf("frame exceeds %d byte limit", MaxFrameBytes)
	}
	return data, nil
}

func ValidateFrame(frame Frame) error {
	switch strings.TrimSpace(frame.Type) {
	case FrameAuthChallenge:
		return requireFields(map[string]string{"challenge": frame.Challenge})
	case FrameAuthResponse:
		return requireFields(map[string]string{
			"device_id": frame.DeviceID,
			"challenge": frame.Challenge,
			"signature": frame.Signature,
		})
	case FrameAuthOK:
		return nil
	case FrameAuthError:
		return requireFields(map[string]string{"error": frame.Error})
	case FrameRequest:
		if err := requireFields(map[string]string{
			"request_id": frame.RequestID,
			"action":     frame.Action,
			"trace_id":   frame.TraceID,
		}); err != nil {
			return err
		}
		if len(frame.Params) == 0 {
			return errors.New("params is required")
		}
		return nil
	case FrameResponse:
		if err := requireFields(map[string]string{
			"request_id": frame.RequestID,
			"status":     frame.Status,
		}); err != nil {
			return err
		}
		if frame.Status != "success" && frame.Status != "error" {
			return fmt.Errorf("invalid response status: %s", frame.Status)
		}
		return nil
	case FrameStreamStart:
		return requireFields(map[string]string{
			"request_id": frame.RequestID,
			"action":     frame.Action,
			"trace_id":   frame.TraceID,
		})
	case FrameStreamEvent:
		return requireFields(map[string]string{
			"request_id": frame.RequestID,
			"event":      frame.Event,
		})
	case FrameStreamEnd:
		return requireFields(map[string]string{
			"request_id": frame.RequestID,
			"status":     frame.Status,
		})
	case FrameCancel:
		return requireFields(map[string]string{
			"request_id": frame.RequestID,
			"trace_id":   frame.TraceID,
		})
	default:
		return fmt.Errorf("unsupported frame type: %s", strings.TrimSpace(frame.Type))
	}
}

func RequestRawParams(frame Frame) json.RawMessage {
	if len(frame.Params) == 0 {
		return json.RawMessage(`{}`)
	}
	return frame.Params
}

func requireFields(fields map[string]string) error {
	for name, value := range fields {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	return nil
}
