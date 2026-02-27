package llm

import (
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
