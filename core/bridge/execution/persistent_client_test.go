package execution

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestWriteFrame(t *testing.T) {
	var buffer bytes.Buffer
	req := request{
		Action:    "PING",
		Params:    map[string]any{"ok": true},
		TraceID:   "trace-1",
		RequestID: "req-1",
	}

	if err := writeFrame(&buffer, req); err != nil {
		t.Fatalf("writeFrame returned error: %v", err)
	}

	raw := buffer.Bytes()
	if len(raw) < 4 {
		t.Fatalf("encoded frame too short: %d", len(raw))
	}
	length := binary.BigEndian.Uint32(raw[:4])
	if int(length) != len(raw)-4 {
		t.Fatalf("unexpected payload length: got %d want %d", length, len(raw)-4)
	}

	var decoded request
	if err := json.Unmarshal(raw[4:], &decoded); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if decoded.Action != "PING" || decoded.RequestID != "req-1" {
		t.Fatalf("unexpected request payload: %+v", decoded)
	}
}

func TestReadFrame(t *testing.T) {
	frame := encodeFrame(t, response{
		Status:    "success",
		Payload:   map[string]any{"message": "PONG"},
		Error:     "",
		RequestID: "req-2",
	})

	var decoded response
	if err := readFrame(bytes.NewReader(frame), &decoded); err != nil {
		t.Fatalf("readFrame returned error: %v", err)
	}
	if decoded.RequestID != "req-2" {
		t.Fatalf("unexpected request id: got %q want %q", decoded.RequestID, "req-2")
	}
	if decoded.Payload["message"] != "PONG" {
		t.Fatalf("unexpected payload: %+v", decoded.Payload)
	}
}

func TestInvalidFrameLength(t *testing.T) {
	var raw bytes.Buffer
	if err := binary.Write(&raw, binary.BigEndian, uint32(maxFrameBytes+1)); err != nil {
		t.Fatalf("write length: %v", err)
	}
	raw.WriteString("{}")

	var decoded response
	err := readFrame(bytes.NewReader(raw.Bytes()), &decoded)
	if err == nil {
		t.Fatal("expected invalid frame length error")
	}
	if got := err.Error(); got == "" || !bytes.Contains([]byte(got), []byte("invalid frame length")) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAutoRestart(t *testing.T) {
	client := newTestPersistentClient(t, persistentHelperOptions{exitAfterOne: true})
	defer func() {
		_ = client.Close()
	}()

	first, err := client.Call(context.Background(), "PING", nil, "trace-first")
	if err != nil {
		t.Fatalf("first call returned error: %v", err)
	}
	if first["trace_id"] != "trace-first" {
		t.Fatalf("unexpected first payload: %+v", first)
	}
	waitForProcessExit(t, client)

	second, err := client.Call(context.Background(), "PING", nil, "trace-second")
	if err != nil {
		t.Fatalf("second call returned error: %v", err)
	}
	if second["trace_id"] != "trace-second" {
		t.Fatalf("unexpected second payload: %+v", second)
	}
}

func TestPersistentProtocolUnsupportedReturnsError(t *testing.T) {
	t.Setenv("GO_PERSISTENT_HELPER_ONESHOT_ONLY", "1")

	client := newTestPersistentClient(t, persistentHelperOptions{})
	defer func() {
		_ = client.Close()
	}()

	_, err := client.Call(context.Background(), "PING", nil, "trace-unsupported")
	if err == nil {
		t.Fatal("expected persistent protocol unsupported error")
	}
	if !errors.Is(err, errPersistentProtocolUnsupported) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContextTimeout(t *testing.T) {
	client := newTestPersistentClient(t, persistentHelperOptions{})
	defer func() {
		_ = client.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.Call(ctx, "BLOCK", nil, "trace-block")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("unexpected timeout error: %v", err)
	}

	payload, err := client.Call(context.Background(), "PING", nil, "trace-after-timeout")
	if err != nil {
		t.Fatalf("call after timeout returned error: %v", err)
	}
	if payload["trace_id"] != "trace-after-timeout" {
		t.Fatalf("unexpected payload after timeout: %+v", payload)
	}
}

func TestSerialExecution(t *testing.T) {
	client := newTestPersistentClient(t, persistentHelperOptions{})
	defer func() {
		_ = client.Close()
	}()

	const totalCalls = 8
	errCh := make(chan error, totalCalls)
	var wg sync.WaitGroup

	for idx := 0; idx < totalCalls; idx++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			traceID := fmt.Sprintf("trace-%d", i)
			payload, err := client.Call(context.Background(), "PING", nil, traceID)
			if err != nil {
				errCh <- err
				return
			}
			if payload["trace_id"] != traceID {
				errCh <- fmt.Errorf("unexpected payload trace_id: got %v want %s", payload["trace_id"], traceID)
			}
		}(idx)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent call failed: %v", err)
		}
	}
}

func TestPersistentClientRejectsMismatchedResponseRequestID(t *testing.T) {
	client := newTestPersistentClient(t, persistentHelperOptions{wrongRequestID: true})
	defer func() {
		_ = client.Close()
	}()

	_, err := client.Call(context.Background(), "PING", nil, "trace-mismatch")
	if err == nil {
		t.Fatal("expected request_id mismatch error")
	}
	if !strings.Contains(err.Error(), "unexpected native response request_id") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewClientWithOptionsUsesPersistentMode(t *testing.T) {
	client := NewClientWithOptions(ClientOptions{Persistent: true})
	if _, ok := client.(*PersistentNativeClient); !ok {
		t.Fatalf("expected persistent client, got %T", client)
	}
}

func TestPersistentClientHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_PERSISTENT_HELPER") != "1" {
		return
	}
	if os.Getenv("GO_PERSISTENT_HELPER_ONESHOT_ONLY") == "1" {
		runOneshotCompatHelper()
	}

	exitAfterOne := os.Getenv("GO_PERSISTENT_HELPER_EXIT_AFTER_ONE") == "1"
	wrongRequestID := os.Getenv("GO_PERSISTENT_HELPER_WRONG_REQUEST_ID") == "1"
	for {
		var req request
		err := readFrame(os.Stdin, &req)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				os.Exit(0)
			}
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}

		switch req.Action {
		case "BLOCK":
			time.Sleep(30 * time.Second)
		case "CRASH":
			os.Exit(3)
		default:
			requestID := req.RequestID
			if wrongRequestID && req.TraceID != "bridge-exec-persistent-handshake" {
				requestID = requestID + "-wrong"
			}
			resp := response{
				Status: "success",
				Payload: map[string]any{
					"message":  "PONG",
					"trace_id": req.TraceID,
					"action":   req.Action,
				},
				RequestID: requestID,
			}
			if err := writeFrame(os.Stdout, resp); err != nil {
				_, _ = fmt.Fprintln(os.Stderr, err)
				os.Exit(4)
			}
			if exitAfterOne && req.TraceID != "bridge-exec-persistent-handshake" {
				os.Exit(0)
			}
		}
	}
}

func runOneshotCompatHelper() {
	_ = json.NewEncoder(os.Stdout).Encode(response{
		Status: "success",
		Payload: map[string]any{
			"mode": "oneshot",
		},
	})
	os.Exit(0)
}

type persistentHelperOptions struct {
	exitAfterOne   bool
	wrongRequestID bool
}

func newTestPersistentClient(t *testing.T, opts persistentHelperOptions) *PersistentNativeClient {
	t.Helper()

	client := NewPersistentNativeClient()
	client.commandFactory = func(_ string) *exec.Cmd {
		cmd := exec.Command(os.Args[0], "-test.run=^TestPersistentClientHelperProcess$")
		cmd.Env = append(os.Environ(), "GO_WANT_PERSISTENT_HELPER=1")
		if opts.exitAfterOne {
			cmd.Env = append(cmd.Env, "GO_PERSISTENT_HELPER_EXIT_AFTER_ONE=1")
		}
		if opts.wrongRequestID {
			cmd.Env = append(cmd.Env, "GO_PERSISTENT_HELPER_WRONG_REQUEST_ID=1")
		}
		return cmd
	}
	return client
}

func encodeFrame(t *testing.T, payload any) []byte {
	t.Helper()

	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	frame := make([]byte, 4+len(encoded))
	binary.BigEndian.PutUint32(frame[:4], uint32(len(encoded)))
	copy(frame[4:], encoded)
	return frame
}

func waitForProcessExit(t *testing.T, client *PersistentNativeClient) {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		client.mu.Lock()
		client.reapExitedProcessLocked()
		exited := client.cmd == nil
		client.mu.Unlock()
		if exited {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("persistent helper did not exit in time")
}
