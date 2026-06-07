package tools

import (
	"context"
	"encoding/json"
	"testing"
)

type mockPersistentExecutionClient struct {
	callFunc func(ctx context.Context, action string, params map[string]any, traceID string) (map[string]any, error)
}

func (m mockPersistentExecutionClient) Call(
	ctx context.Context,
	action string,
	params map[string]any,
	traceID string,
) (map[string]any, error) {
	return m.callFunc(ctx, action, params, traceID)
}

func (mockPersistentExecutionClient) SupportsPersistentSessions() bool {
	return true
}

func TestBashExecToolExecuteSuccess(t *testing.T) {
	tool := NewBashExecTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, params map[string]any, traceID string) (map[string]any, error) {
			if action != "BASH_EXEC" {
				t.Fatalf("unexpected action: got %q want %q", action, "BASH_EXEC")
			}
			if traceID != "trace-bash-1" {
				t.Fatalf("unexpected trace id: got %q want %q", traceID, "trace-bash-1")
			}
			if params["command"] != "pwd" || params["max_output_chars"] != 512 {
				t.Fatalf("unexpected params: %+v", params)
			}
			return map[string]any{
				"stdout": "/tmp\n",
				"stderr": "",
			}, nil
		},
	})

	output, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"command":" pwd ","max_output_chars":512}`),
		"trace-bash-1",
	)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if output != "/tmp\n" {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestBashExecToolExecuteNormalizesOneShotHintParams(t *testing.T) {
	tool := NewBashExecTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, params map[string]any, traceID string) (map[string]any, error) {
			if action != "BASH_EXEC" {
				t.Fatalf("unexpected action: got %q want %q", action, "BASH_EXEC")
			}
			if traceID != "trace-bash-hints" {
				t.Fatalf("unexpected trace id: got %q want %q", traceID, "trace-bash-hints")
			}
			if params["command"] != "python --version 2>/dev/null || python3 --version 2>/dev/null" {
				t.Fatalf("unexpected command: %+v", params)
			}
			if params["login"] != true || params["timeout_ms"] != 10000 || params["max_output_chars"] != 2000 {
				t.Fatalf("unexpected one-shot params: %+v", params)
			}
			for _, key := range []string{"interactive", "session_id", "tty", "yield_time_ms"} {
				if _, exists := params[key]; exists {
					t.Fatalf("unexpected forwarded hint %q in params: %+v", key, params)
				}
			}
			return map[string]any{
				"stdout": "Python 3.12.3\n",
				"stderr": "",
			}, nil
		},
	})

	output, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"command":"python --version 2>/dev/null || python3 --version 2>/dev/null","interactive":false,"login":true,"max_output_chars":2000,"session_id":"","timeout_ms":10000,"tty":false,"yield_time_ms":1000}`),
		"trace-bash-hints",
	)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if output != "Python 3.12.3\n" {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestBashExecToolExecuteRequiresCommand(t *testing.T) {
	tool := NewBashExecTool(mockExecutionClient{
		callFunc: func(_ context.Context, _ string, _ map[string]any, _ string) (map[string]any, error) {
			t.Fatal("execution client should not be called")
			return nil, nil
		},
	})

	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"command":" "}`), "trace-bash-2"); err == nil {
		t.Fatal("expected command validation error")
	}
}

func TestBashExecToolExecuteRejectsZeroOutputLimit(t *testing.T) {
	tool := NewBashExecTool(mockExecutionClient{
		callFunc: func(_ context.Context, _ string, _ map[string]any, _ string) (map[string]any, error) {
			t.Fatal("execution client should not be called")
			return nil, nil
		},
	})

	if _, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"command":"pwd","max_output_chars":0}`),
		"trace-bash-3",
	); err == nil {
		t.Fatal("expected max_output_chars validation error")
	}
}

func TestBashExecToolExecuteRejectsInteractiveWithoutPersistentClient(t *testing.T) {
	tool := NewBashExecTool(mockExecutionClient{
		callFunc: func(_ context.Context, _ string, _ map[string]any, _ string) (map[string]any, error) {
			t.Fatal("execution client should not be called")
			return nil, nil
		},
	})

	if _, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"command":"pwd","interactive":true}`),
		"trace-bash-4",
	); err == nil {
		t.Fatal("expected native_persistent validation error")
	}
}

func TestBashExecToolExecuteInteractiveSuccess(t *testing.T) {
	tool := NewBashExecTool(mockPersistentExecutionClient{
		callFunc: func(_ context.Context, action string, params map[string]any, traceID string) (map[string]any, error) {
			if action != "BASH_EXEC" {
				t.Fatalf("unexpected action: got %q want %q", action, "BASH_EXEC")
			}
			if traceID != "trace-bash-5" {
				t.Fatalf("unexpected trace id: got %q want %q", traceID, "trace-bash-5")
			}
			if params["command"] != "echo hi" || params["interactive"] != true || params["session_id"] != "sess-1" {
				t.Fatalf("unexpected params: %+v", params)
			}
			if params["yield_time_ms"] != 25 {
				t.Fatalf("unexpected params: %+v", params)
			}
			return map[string]any{
				"session_id": "sess-1",
				"stdout":     "hi\n",
				"stderr":     "",
				"running":    true,
			}, nil
		},
	})

	output, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"command":" echo hi ","interactive":true,"session_id":"sess-1","yield_time_ms":25}`),
		"trace-bash-5",
	)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if payload["session_id"] != "sess-1" || payload["running"] != true {
		t.Fatalf("unexpected output payload: %+v", payload)
	}
	if _, exists := payload["exit_code"]; exists {
		t.Fatalf("unexpected exit_code for running session: %+v", payload)
	}
}

func TestBashExecToolExecuteRejectsInvalidParamCombination(t *testing.T) {
	tool := NewBashExecTool(mockPersistentExecutionClient{
		callFunc: func(_ context.Context, _ string, _ map[string]any, _ string) (map[string]any, error) {
			t.Fatal("execution client should not be called")
			return nil, nil
		},
	})

	cases := []json.RawMessage{
		json.RawMessage(`{"command":"pwd","session_id":"sess-1"}`),
		json.RawMessage(`{"command":"pwd","interactive":true,"login":true}`),
		json.RawMessage(`{"command":"pwd","interactive":true,"timeout_ms":100}`),
		json.RawMessage(`{"command":"pwd","tty":true}`),
	}
	for _, args := range cases {
		if _, err := tool.Execute(context.Background(), args, "trace-bash-6"); err == nil {
			t.Fatalf("expected validation error for args: %s", string(args))
		}
	}
}

func TestBashExecToolSchemaBlocksCommonInteractiveMisuse(t *testing.T) {
	tool := NewBashExecTool(nil)
	raw := tool.Parameters()
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("decode schema: %v", err)
	}

	allOf, ok := schema["allOf"].([]any)
	if !ok || len(allOf) < 2 {
		t.Fatalf("expected schema allOf constraints, got: %+v", schema["allOf"])
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected schema properties, got: %+v", schema["properties"])
	}
	if _, exists := properties["tty"]; exists {
		t.Fatalf("tty is not supported and should not be exposed to the model: %+v", properties["tty"])
	}
}
