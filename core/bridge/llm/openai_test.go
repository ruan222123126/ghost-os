package llm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 验证已知 finish_reason 的严格映射路径。
func TestOpenAIToCompletionResponseKnownFinishReasons(t *testing.T) {
	testCases := []struct {
		name   string
		reason string
		want   FinishReason
	}{
		{
			name:   "stop",
			reason: "stop",
			want:   FinishStop,
		},
		{
			name:   "tool calls",
			reason: "tool_calls",
			want:   FinishToolCalls,
		},
		{
			name:   "length",
			reason: "length",
			want:   FinishLength,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := openAIToCompletionResponse(openAIResponse{
				Choices: []openAIChoice{
					{
						Message: openAIMessage{
							Role:    "assistant",
							Content: "ok",
						},
						FinishReason: tc.reason,
					},
				},
			})
			if err != nil {
				t.Fatalf("openAIToCompletionResponse returned error: %v", err)
			}
			if resp.FinishReason != tc.want {
				t.Fatalf("unexpected finish reason: got %q want %q", resp.FinishReason, tc.want)
			}
		})
	}
}

// 验证空 finish_reason 在有文本时回落为 stop。
func TestOpenAIToCompletionResponseEmptyFinishReasonFallsBackToStop(t *testing.T) {
	resp, err := openAIToCompletionResponse(openAIResponse{
		Choices: []openAIChoice{
			{
				Message: openAIMessage{
					Role:    "assistant",
					Content: "ok",
				},
				FinishReason: "",
			},
		},
	})
	if err != nil {
		t.Fatalf("openAIToCompletionResponse returned error: %v", err)
	}
	if resp.FinishReason != FinishStop {
		t.Fatalf("unexpected finish reason: got %q want %q", resp.FinishReason, FinishStop)
	}
}

// 验证空 finish_reason 且包含 tool_calls 时回落为 tool_calls。
func TestOpenAIToCompletionResponseEmptyFinishReasonFallsBackToToolCalls(t *testing.T) {
	resp, err := openAIToCompletionResponse(openAIResponse{
		Choices: []openAIChoice{
			{
				Message: openAIMessage{
					Role: "assistant",
					ToolCalls: []openAIToolCall{
						{
							ID:   "call-1",
							Type: "function",
							Function: openAIFunctionCall{
								Name:      "echo",
								Arguments: `{"text":"hello"}`,
							},
						},
					},
				},
				FinishReason: "",
			},
		},
	})
	if err != nil {
		t.Fatalf("openAIToCompletionResponse returned error: %v", err)
	}
	if resp.FinishReason != FinishToolCalls {
		t.Fatalf("unexpected finish reason: got %q want %q", resp.FinishReason, FinishToolCalls)
	}
	if len(resp.Message.ToolCalls) != 1 {
		t.Fatalf("unexpected tool calls count: got %d want %d", len(resp.Message.ToolCalls), 1)
	}
}

// 验证空 tool_call.arguments 不被静默改写为 {}，交由上层做无效调用处理。
func TestOpenAIToCompletionResponsePreservesEmptyToolCallArguments(t *testing.T) {
	resp, err := openAIToCompletionResponse(openAIResponse{
		Choices: []openAIChoice{
			{
				Message: openAIMessage{
					Role: "assistant",
					ToolCalls: []openAIToolCall{
						{
							ID:   "call-1",
							Type: "function",
							Function: openAIFunctionCall{
								Name:      "script_exec",
								Arguments: "",
							},
						},
					},
				},
				FinishReason: "tool_calls",
			},
		},
	})
	if err != nil {
		t.Fatalf("openAIToCompletionResponse returned error: %v", err)
	}
	if len(resp.Message.ToolCalls) != 1 {
		t.Fatalf("unexpected tool calls count: got %d want %d", len(resp.Message.ToolCalls), 1)
	}
	if strings.TrimSpace(string(resp.Message.ToolCalls[0].Arguments)) != "" {
		t.Fatalf("unexpected arguments: got %q want empty", string(resp.Message.ToolCalls[0].Arguments))
	}
}

// 验证未知 finish_reason 走错误分支，不允许静默降级。
func TestOpenAIToCompletionResponseUnknownFinishReasonReturnsError(t *testing.T) {
	_, err := openAIToCompletionResponse(openAIResponse{
		Choices: []openAIChoice{
			{
				Message: openAIMessage{
					Role:    "assistant",
					Content: "ok",
				},
				FinishReason: "unexpected",
			},
		},
	})
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !strings.Contains(err.Error(), "unsupported openai finish_reason") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestToOpenAIRequestBuildsToolImageContentParts(t *testing.T) {
	tempDir := t.TempDir()
	imagePath := filepath.Join(tempDir, "tool-shot.png")
	if err := os.WriteFile(imagePath, []byte("fake-tool-png"), 0o600); err != nil {
		t.Fatalf("write temp image: %v", err)
	}

	request, err := toOpenAIRequest("gpt-4o", CompletionRequest{
		Messages: []Message{
			{
				Role: RoleAssistant,
				ToolCalls: []ToolCall{
					{
						ID:        "call-1",
						Name:      "browser_action",
						Arguments: json.RawMessage(`{"action":"screenshot"}`),
					},
				},
			},
			{
				Role:       RoleTool,
				ToolCallID: "call-1",
				Text:       `{"status":"success","tool":"browser_action"}`,
				Content: []ContentPart{
					{
						Type: ContentTypeImage,
						Image: &ImageContent{
							Path:     imagePath,
							MimeType: "image/png",
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("toOpenAIRequest returned error: %v", err)
	}

	if len(request.Messages) != 2 {
		t.Fatalf("unexpected message count: got %d want %d", len(request.Messages), 2)
	}
	encoded, err := json.Marshal(request.Messages[1].Content)
	if err != nil {
		t.Fatalf("marshal tool content: %v", err)
	}
	if !strings.Contains(string(encoded), `"type":"image_url"`) {
		t.Fatalf("expected tool image_url block, got: %s", string(encoded))
	}
	if !strings.Contains(string(encoded), `\"status\":\"success\"`) {
		t.Fatalf("expected tool envelope text in content, got: %s", string(encoded))
	}
}
