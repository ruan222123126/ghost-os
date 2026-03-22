package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/tools"
)

type fakeCompleter struct {
	responses []*llm.CompletionResponse
	requests  []llm.CompletionRequest
}

func newFakeCompleter(responses ...*llm.CompletionResponse) *fakeCompleter {
	return &fakeCompleter{responses: responses}
}

// fakeCompleter 通过预置响应驱动 Agent 循环，避免真实网络依赖。
func (f *fakeCompleter) Complete(_ context.Context, request llm.CompletionRequest) (*llm.CompletionResponse, error) {
	f.requests = append(f.requests, cloneCompletionRequest(request))
	if len(f.responses) == 0 {
		return nil, errors.New("unexpected complete call")
	}

	response := f.responses[0]
	f.responses = f.responses[1:]
	return response, nil
}

type fakeStreamingCompleter struct {
	completeResponses []*llm.CompletionResponse
	streamResponses   []*llm.CompletionResponse
	completeRequests  []llm.CompletionRequest
	streamRequests    []llm.CompletionRequest
	streamDeltas      [][]llm.LLMDelta
}

func (f *fakeStreamingCompleter) Complete(_ context.Context, request llm.CompletionRequest) (*llm.CompletionResponse, error) {
	f.completeRequests = append(f.completeRequests, cloneCompletionRequest(request))
	if len(f.completeResponses) == 0 {
		return nil, errors.New("unexpected complete call")
	}
	response := f.completeResponses[0]
	f.completeResponses = f.completeResponses[1:]
	return response, nil
}

func (f *fakeStreamingCompleter) CompleteStream(ctx context.Context, request llm.CompletionRequest, sink llm.LLMStreamSink) (*llm.CompletionResponse, error) {
	f.streamRequests = append(f.streamRequests, cloneCompletionRequest(request))
	if len(f.streamResponses) == 0 {
		return nil, errors.New("unexpected complete stream call")
	}
	if len(f.streamDeltas) > 0 {
		deltas := f.streamDeltas[0]
		f.streamDeltas = f.streamDeltas[1:]
		for _, delta := range deltas {
			if err := sink.OnDelta(ctx, delta); err != nil {
				return nil, err
			}
		}
	}
	response := f.streamResponses[0]
	f.streamResponses = f.streamResponses[1:]
	return response, nil
}

func cloneCompletionRequest(request llm.CompletionRequest) llm.CompletionRequest {
	clonedTools := make([]llm.ToolDef, len(request.Tools))
	for i, tool := range request.Tools {
		clonedTools[i] = llm.ToolDef{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  cloneRawJSON(tool.Parameters),
			Semantics:   tool.Semantics,
		}
	}

	return llm.CompletionRequest{
		Messages:          llm.CloneMessages(request.Messages),
		Tools:             clonedTools,
		ConversationState: request.ConversationState,
	}
}

func cloneRawJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}

	out := make([]byte, len(raw))
	copy(out, raw)
	return out
}

type fakeToolCatalog struct {
	defs      []llm.ToolDef
	toolByKey map[string]tools.Tool
}

func (f *fakeToolCatalog) ToolDefs() []llm.ToolDef {
	out := make([]llm.ToolDef, len(f.defs))
	for i, tool := range f.defs {
		out[i] = llm.ToolDef{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  cloneRawJSON(tool.Parameters),
			Semantics:   tool.Semantics,
		}
	}
	return out
}

func (f *fakeToolCatalog) Get(name string) tools.Tool {
	if f.toolByKey == nil {
		return nil
	}
	return f.toolByKey[name]
}

func newFakeToolCatalog(testTools ...*fakeTool) *fakeToolCatalog {
	if len(testTools) == 0 {
		return &fakeToolCatalog{}
	}

	defs := make([]llm.ToolDef, 0, len(testTools))
	toolByKey := make(map[string]tools.Tool, len(testTools))
	for _, tool := range testTools {
		if tool == nil {
			continue
		}
		defs = append(defs, llm.ToolDef{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  cloneRawJSON(tool.Parameters()),
			Semantics:   tool.ToolSemantics(),
		})
		toolByKey[tool.Name()] = tool
	}

	return &fakeToolCatalog{defs: defs, toolByKey: toolByKey}
}

type fakeTool struct {
	name      string
	execute   func(context.Context, json.RawMessage) (string, error)
	executeV2 func(context.Context, json.RawMessage, string) (string, error)
	interpret func(string) tools.ExecuteMeta
	semantics llm.ToolSemantics
	callCount int
	lastArgs  json.RawMessage
	lastTrace string
}

func (f *fakeTool) Name() string {
	return f.name
}

func (f *fakeTool) Description() string {
	return "fake tool"
}

func (f *fakeTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object"}`)
}

func (f *fakeTool) ToolSemantics() llm.ToolSemantics {
	return f.semantics
}

func (f *fakeTool) Execute(ctx context.Context, argsJSON json.RawMessage, traceID string) (string, error) {
	f.callCount++
	f.lastArgs = cloneRawJSON(argsJSON)
	f.lastTrace = traceID
	if f.executeV2 != nil {
		return f.executeV2(ctx, argsJSON, traceID)
	}
	if f.execute == nil {
		return "", nil
	}
	return f.execute(ctx, argsJSON)
}

func (f *fakeTool) InterpretResult(output string) tools.ExecuteMeta {
	if f.interpret == nil {
		return tools.ExecuteMeta{}
	}
	return f.interpret(output)
}

func newStaticTool(name string, output string) *fakeTool {
	return &fakeTool{
		name: name,
		execute: func(_ context.Context, _ json.RawMessage) (string, error) {
			return output, nil
		},
	}
}

func newErrorTool(name string, err error) *fakeTool {
	return &fakeTool{
		name: name,
		execute: func(_ context.Context, _ json.RawMessage) (string, error) {
			return "", err
		},
	}
}

func newAwaitingHumanTool(name string, questionID string, prompt string) *fakeTool {
	output := fmt.Sprintf(`{"status":"awaiting_human","question_id":%q,"prompt":%q}`, questionID, prompt)
	return &fakeTool{
		name: name,
		execute: func(_ context.Context, _ json.RawMessage) (string, error) {
			return output, nil
		},
		interpret: func(string) tools.ExecuteMeta {
			return tools.ExecuteMeta{
				AwaitingHuman: &tools.AwaitingHumanSignal{
					QuestionID: questionID,
					Prompt:     prompt,
				},
			}
		},
	}
}

func newAssistantResponse(finishReason llm.FinishReason, text string, toolCalls ...llm.ToolCall) *llm.CompletionResponse {
	return &llm.CompletionResponse{
		Message: llm.Message{
			Role:      llm.RoleAssistant,
			Text:      text,
			ToolCalls: toolCalls,
		},
		FinishReason: finishReason,
	}
}

func newStopResponse(text string) *llm.CompletionResponse {
	return newAssistantResponse(llm.FinishStop, text)
}

func newLengthResponse(text string) *llm.CompletionResponse {
	return newAssistantResponse(llm.FinishLength, text)
}

func newToolCallsResponse(toolCalls ...llm.ToolCall) *llm.CompletionResponse {
	return newAssistantResponse(llm.FinishToolCalls, "", toolCalls...)
}

func newToolCall(id string, name string, args string) llm.ToolCall {
	return llm.ToolCall{
		ID:        id,
		Name:      name,
		Arguments: json.RawMessage(args),
	}
}

type toolResultPayload struct {
	Status  string `json:"status"`
	Tool    string `json:"tool"`
	TraceID string `json:"trace_id"`
	Output  string `json:"output"`
	Error   string `json:"error"`
}

// decodeToolResult 用于校验 tool_result 的 JSON envelope 结构。
func decodeToolResult(t *testing.T, raw string) toolResultPayload {
	t.Helper()

	var result toolResultPayload
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("decode tool result JSON: %v, raw=%s", err, raw)
	}
	return result
}

func lastMessage(t *testing.T, request llm.CompletionRequest) llm.Message {
	t.Helper()

	if len(request.Messages) == 0 {
		t.Fatal("request has no messages")
	}
	return request.Messages[len(request.Messages)-1]
}

func newTestAgent(completer Completer, catalog *fakeToolCatalog, maxTurns int) *Agent {
	return NewAgent(completer, catalog, "system prompt", maxTurns)
}

func newRecordingEventSink() *recordingEventSink {
	return &recordingEventSink{}
}

type fakeGraphQLTextExecutor struct {
	results []tools.GraphQLTextExecutionResult
	errors  []error
	texts   []string
	traces  []string
}

func (f *fakeGraphQLTextExecutor) Execute(
	_ context.Context,
	text string,
	traceID string,
) (tools.GraphQLTextExecutionResult, error) {
	f.texts = append(f.texts, text)
	f.traces = append(f.traces, traceID)
	if len(f.results) == 0 {
		return tools.GraphQLTextExecutionResult{}, errors.New("unexpected graphql text execute call")
	}
	result := f.results[0]
	f.results = f.results[1:]
	var err error
	if len(f.errors) > 0 {
		err = f.errors[0]
		f.errors = f.errors[1:]
	}
	return result, err
}

type recordingEventSink struct {
	events []streaming.Event
}

func (r *recordingEventSink) Emit(_ context.Context, event streaming.Event) (streaming.Event, error) {
	r.events = append(r.events, event)
	return event, nil
}
