package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

var (
	traceCounter         uint64
	errRequestBodyTooBig = errors.New("request body too large")
)

func decodeJSONBody(w http.ResponseWriter, r *http.Request, maxBytes int64, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return errRequestBodyTooBig
		}
		if errors.Is(err, io.EOF) {
			return errors.New("invalid JSON body: empty request body")
		}
		return fmt.Errorf("invalid JSON body: %w", err)
	}

	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err == nil {
		return errors.New("invalid JSON body: multiple JSON values are not allowed")
	} else if !errors.Is(err, io.EOF) {
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	return nil
}

func decodeStatusCode(err error) int {
	if errors.Is(err, errRequestBodyTooBig) {
		return http.StatusRequestEntityTooLarge
	}
	return http.StatusBadRequest
}

func resolveTraceID(candidate string, r *http.Request) string {
	if traceID := strings.TrimSpace(candidate); traceID != "" {
		return traceID
	}
	if traceID := strings.TrimSpace(r.Header.Get("X-Trace-ID")); traceID != "" {
		return traceID
	}
	return nextTraceID()
}

func nextTraceID() string {
	sequence := atomic.AddUint64(&traceCounter, 1)
	return fmt.Sprintf("bridge-%d-%d", time.Now().UnixMilli(), sequence)
}

func writeSuccess(w http.ResponseWriter, code int, payload any, traceID string) {
	writeEnvelope(w, code, apiResponse{
		Status:  "success",
		Payload: payload,
		Error:   "",
	}, traceID)
}

func writeError(w http.ResponseWriter, code int, message string, traceID string) {
	writeEnvelope(w, code, apiResponse{
		Status:  "error",
		Payload: map[string]any{},
		Error:   message,
	}, traceID)
}

func writeEnvelope(w http.ResponseWriter, code int, response apiResponse, traceID string) {
	w.Header().Set("Content-Type", "application/json")
	if traceID != "" {
		w.Header().Set("X-Trace-ID", traceID)
	}
	w.WriteHeader(code)
	if response.Payload == nil {
		response.Payload = map[string]any{}
	}
	_ = json.NewEncoder(w).Encode(response)
}

func logAction(traceID string, action string, status string, err error) {
	if err != nil {
		log.Printf("trace_id=%s action=%s status=%s error=%v", traceID, action, status, err)
		return
	}
	log.Printf("trace_id=%s action=%s status=%s", traceID, action, status)
}
