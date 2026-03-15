package rss

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"ghost-os/bridge/llm"
)

const (
	testRSSWindowHours         = 24
	testRSSGroupLimit          = 5
	testRSSItemLimit           = 10
	testRSSItemsPerGroup       = 3
	testRSSHighlightsLimit     = 3
	testRSSBuilderMaxTurns     = 6
	testRSSBuildRequestCount   = 3
	testRSSScopedToolCount     = 2
	testRSSSingleToolCallCount = 1
	testRSSReportTimeout       = 5 * time.Second
)

type stubRSSBriefingBuilder struct {
	draft rssBriefingDraft
	err   error
}

func (b stubRSSBriefingBuilder) Build(context.Context, []RSSInboxTopicGroup, RSSBriefingQuery) (rssBriefingDraft, error) {
	return b.draft, b.err
}

type stubRSSReportBuilder struct {
	markdown string
	err      error
}

func (b stubRSSReportBuilder) Build(context.Context, RSSReportResult, RSSBriefingResult, []RSSInboxTopicGroup, RSSReportQuery) (string, error) {
	return b.markdown, b.err
}

type fakeReportCompleter struct {
	responses []*llm.CompletionResponse
	requests  []llm.CompletionRequest
}

func (f *fakeReportCompleter) Complete(_ context.Context, request llm.CompletionRequest) (*llm.CompletionResponse, error) {
	f.requests = append(f.requests, cloneReportCompletionRequest(request))
	if len(f.responses) == 0 {
		return nil, errors.New("unexpected complete call")
	}
	response := f.responses[0]
	f.responses = f.responses[1:]
	return response, nil
}

func cloneReportCompletionRequest(request llm.CompletionRequest) llm.CompletionRequest {
	return llm.CompletionRequest{
		Messages: llm.CloneMessages(request.Messages),
		Tools:    append([]llm.ToolDef(nil), request.Tools...),
	}
}

type fakeReportTool struct {
	name      string
	output    string
	callCount int
	lastArgs  json.RawMessage
}

func (f *fakeReportTool) Name() string { return f.name }

func (f *fakeReportTool) Description() string { return "fake report tool" }

func (f *fakeReportTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object"}`)
}

func (f *fakeReportTool) Execute(_ context.Context, argsJSON json.RawMessage, _ string) (string, error) {
	f.callCount++
	f.lastArgs = append([]byte(nil), argsJSON...)
	return f.output, nil
}

func reportToolCallsResponse(calls ...llm.ToolCall) *llm.CompletionResponse {
	return &llm.CompletionResponse{
		Message:      llm.Message{Role: llm.RoleAssistant, ToolCalls: calls},
		FinishReason: llm.FinishToolCalls,
	}
}

func reportStopResponse(text string) *llm.CompletionResponse {
	return &llm.CompletionResponse{
		Message:      llm.Message{Role: llm.RoleAssistant, Text: text},
		FinishReason: llm.FinishStop,
	}
}

func mustMarshalJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return data
}

func requireReportMetadata(t *testing.T, result RSSBriefingResult, now time.Time) {
	t.Helper()
	if result.Report == nil {
		t.Fatal("expected report metadata")
	}
	if result.Report.BriefingID != result.ID {
		t.Fatalf("unexpected report briefing id: got %q want %q", result.Report.BriefingID, result.ID)
	}
	if !strings.Contains(result.Report.Title, now.UTC().Format("2006-01-02 15:04 UTC")) {
		t.Fatalf("expected report title to include generated time, got %q", result.Report.Title)
	}
	if !strings.HasSuffix(result.Report.MarkdownPath, ".md") {
		t.Fatalf("expected markdown path, got %q", result.Report.MarkdownPath)
	}
}

func requireReportToolCallCount(t *testing.T, tool *fakeReportTool, want int) {
	t.Helper()
	if tool.callCount != want {
		t.Fatalf("expected %s to be called %d times, got %d", tool.name, want, tool.callCount)
	}
}

func requireToolDefNames(t *testing.T, defs []llm.ToolDef, want []string) {
	t.Helper()
	if len(defs) != len(want) {
		t.Fatalf("expected %d tool defs, got %d", len(want), len(defs))
	}
	for i, name := range want {
		if defs[i].Name != name {
			t.Fatalf("unexpected tool name at %d: got %q want %q", i, defs[i].Name, name)
		}
	}
}

func requireFileContains(t *testing.T, path, want string) {
	t.Helper()
	requireStringContains(t, readTestFile(t, path), want)
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}
	return string(body)
}

func requireStringContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("expected %q to contain %q", got, want)
	}
}

func requireStringNotContains(t *testing.T, got, want string) {
	t.Helper()
	if strings.Contains(got, want) {
		t.Fatalf("expected %q to exclude %q", got, want)
	}
}
