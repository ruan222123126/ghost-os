package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

type Agent struct {
	client   *llm.Client
	registry *tools.Registry
	history  *History
	maxTurns int
}

func NewAgent(client *llm.Client, registry *tools.Registry, systemPrompt string, maxTurns int) *Agent {
	if maxTurns <= 0 {
		maxTurns = 20
	}

	return &Agent{
		client:   client,
		registry: registry,
		history:  NewHistory(systemPrompt),
		maxTurns: maxTurns,
	}
}

func (a *Agent) Run(ctx context.Context, userMessage string) (string, error) {
	traceID := fmt.Sprintf("agent-%d", time.Now().UnixNano())
	a.history.Append(llm.ChatMessage{
		Role:    "user",
		Content: userMessage,
	})

	for turn := 0; turn < a.maxTurns; turn++ {
		resp, err := a.client.Complete(ctx, a.history.Messages(), a.registry.ToolDefs())
		if err != nil {
			return "", err
		}
		if len(resp.Choices) == 0 {
			return "", errors.New("llm response has no choices")
		}

		choice := resp.Choices[0]
		msg := choice.Message
		if msg.Role == "" {
			msg.Role = "assistant"
		}
		a.history.Append(msg)

		switch choice.FinishReason {
		case "stop":
			return llm.ContentText(msg.Content), nil
		case "tool_calls":
			if len(msg.ToolCalls) == 0 {
				return "", errors.New("finish_reason=tool_calls but tool_calls is empty")
			}
			for _, call := range msg.ToolCalls {
				name := call.Function.Name
				args := call.Function.Arguments
				fmt.Fprintf(os.Stderr, "[%s] tool_call: %s args=%s\n", traceID, name, args)

				tool := a.registry.Get(name)
				if tool == nil {
					a.history.Append(llm.ChatMessage{
						Role:       "tool",
						ToolCallID: call.ID,
						Content:    fmt.Sprintf("tool %q not found", name),
					})
					continue
				}

				result, err := tool.Execute(ctx, args)
				if err != nil {
					result = fmt.Sprintf("tool %q error: %v", name, err)
				}

				a.history.Append(llm.ChatMessage{
					Role:       "tool",
					ToolCallID: call.ID,
					Content:    result,
				})
			}
		default:
			content := strings.TrimSpace(llm.ContentText(msg.Content))
			if content != "" {
				return content, nil
			}
			return "", fmt.Errorf("unsupported finish_reason: %q", choice.FinishReason)
		}
	}

	return "", fmt.Errorf("max turns exceeded: %d", a.maxTurns)
}
