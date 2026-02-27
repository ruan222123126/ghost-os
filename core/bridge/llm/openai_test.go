package llm

import (
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
