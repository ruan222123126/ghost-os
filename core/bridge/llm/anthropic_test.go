package llm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 验证已知 stop_reason 的严格映射路径。
func TestAnthropicToCompletionResponseKnownStopReasons(t *testing.T) {
	testCases := []struct {
		name       string
		stopReason string
		want       FinishReason
	}{
		{
			name:       "end turn",
			stopReason: "end_turn",
			want:       FinishStop,
		},
		{
			name:       "stop sequence",
			stopReason: "stop_sequence",
			want:       FinishStop,
		},
		{
			name:       "tool use",
			stopReason: "tool_use",
			want:       FinishToolCalls,
		},
		{
			name:       "max tokens",
			stopReason: "max_tokens",
			want:       FinishLength,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := anthropicToCompletionResponse(anthropicResponse{
				Content: []anthropicContentBlock{
					{
						Type: "text",
						Text: "ok",
					},
				},
				StopReason: tc.stopReason,
			})
			if err != nil {
				t.Fatalf("anthropicToCompletionResponse returned error: %v", err)
			}
			if resp.FinishReason != tc.want {
				t.Fatalf("unexpected finish reason: got %q want %q", resp.FinishReason, tc.want)
			}
		})
	}
}

// 验证未知 stop_reason 返回错误，避免误判为正常 stop。
func TestAnthropicToCompletionResponseUnknownStopReasonReturnsError(t *testing.T) {
	_, err := anthropicToCompletionResponse(anthropicResponse{
		Content: []anthropicContentBlock{
			{
				Type: "text",
				Text: "ok",
			},
		},
		StopReason: "unexpected",
	})
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !strings.Contains(err.Error(), "unsupported anthropic stop_reason") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestToAnthropicRequestBuildsToolImageContentBlocks(t *testing.T) {
	tempDir := t.TempDir()
	imagePath := filepath.Join(tempDir, "tool-shot.jpg")
	if err := os.WriteFile(imagePath, []byte("fake-tool-jpg"), 0o600); err != nil {
		t.Fatalf("write temp image: %v", err)
	}

	request, err := toAnthropicRequest("claude-3-7-sonnet", 1024, CompletionRequest{
		Messages: []Message{
			{
				Role: RoleAssistant,
				ToolCalls: []ToolCall{
					{
						ID:        "tool-call-1",
						Name:      "screen_action",
						Arguments: json.RawMessage(`{"action":"screenshot"}`),
					},
				},
			},
			{
				Role:       RoleTool,
				ToolCallID: "tool-call-1",
				Text:       `{"status":"success","tool":"screen_action"}`,
				Content: []ContentPart{
					{
						Type: ContentTypeImage,
						Image: &ImageContent{
							Path:     imagePath,
							MimeType: "image/jpeg",
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("toAnthropicRequest returned error: %v", err)
	}
	if len(request.Messages) != 2 {
		t.Fatalf("unexpected message count: got %d want %d", len(request.Messages), 2)
	}
	encoded, err := json.Marshal(request.Messages[1].Content)
	if err != nil {
		t.Fatalf("marshal tool content: %v", err)
	}
	if !strings.Contains(string(encoded), `"type":"tool_result"`) {
		t.Fatalf("expected tool_result block, got: %s", string(encoded))
	}
	if !strings.Contains(string(encoded), `"type":"image"`) {
		t.Fatalf("expected image block in tool_result, got: %s", string(encoded))
	}
}
