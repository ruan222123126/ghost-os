package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"ghost-os/bridge/artifacts"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
)

type computerUseTestCompleter struct {
	outputs []string
	index   int
}

func (c *computerUseTestCompleter) Complete(
	_ context.Context,
	_ llm.CompletionRequest,
) (*llm.CompletionResponse, error) {
	output := c.outputs[c.index]
	c.index++
	return &llm.CompletionResponse{
		Message:      llm.Message{Text: output},
		FinishReason: llm.FinishStop,
	}, nil
}

func TestComputerUseExecutePersistsAwaitingRun(t *testing.T) {
	store, err := artifacts.NewSessionArtifactStore(t.TempDir())
	if err != nil {
		t.Fatalf("new artifact store: %v", err)
	}
	tool := NewComputerUseTool(
		mockExecutionClient{callFunc: testComputerUseExecutionClient(t)},
		&computerUseTestCompleter{outputs: []string{
			`{"thought":"need otp","action":{"type":"call_user","prompt":"Enter the OTP"}}`,
		}},
		store,
	).(*ComputerUseTool)
	sess := session.NewSession("")
	ctx := WithToolCallID(WithSession(context.Background(), sess), "tool-call-1")

	output, err := tool.Execute(ctx, json.RawMessage(`{"goal":"finish login"}`), "trace-computer-use")
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	meta := tool.InterpretResult(output)
	if meta.AwaitingHuman == nil {
		t.Fatalf("expected awaiting human meta, got %+v", meta)
	}
	if len(sess.PendingQuestions) != 1 || len(sess.PendingComputerUseRuns) != 1 {
		t.Fatalf("expected pending question and run, got %+v %+v", sess.PendingQuestions, sess.PendingComputerUseRuns)
	}
}

func TestComputerUseResumeFromHumanAnswer(t *testing.T) {
	store, err := artifacts.NewSessionArtifactStore(t.TempDir())
	if err != nil {
		t.Fatalf("new artifact store: %v", err)
	}
	tool := NewComputerUseTool(
		mockExecutionClient{callFunc: testComputerUseExecutionClient(t)},
		&computerUseTestCompleter{outputs: []string{
			`{"thought":"need otp","action":{"type":"call_user","prompt":"Enter the OTP"}}`,
			`{"thought":"done","action":{"type":"finished"}}`,
		}},
		store,
	).(*ComputerUseTool)
	sess := session.NewSession("")
	ctx := WithToolCallID(WithSession(context.Background(), sess), "tool-call-1")

	if _, err := tool.Execute(ctx, json.RawMessage(`{"goal":"finish login"}`), "trace-computer-use"); err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	var questionID string
	for id := range sess.PendingQuestions {
		questionID = id
	}

	output, meta, handled, err := tool.ResumeFromHumanAnswer(ctx, questionID, "123456", "trace-resume")
	if err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	if !handled || meta.AwaitingHuman != nil {
		t.Fatalf("unexpected resume meta: handled=%t meta=%+v", handled, meta)
	}
	var result struct {
		Status    string `json:"status"`
		Completed bool   `json:"completed"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Status != "success" || !result.Completed {
		t.Fatalf("unexpected result payload: %s", output)
	}
	if len(sess.PendingComputerUseRuns) != 0 {
		t.Fatalf("expected pending runs to be cleared, got %+v", sess.PendingComputerUseRuns)
	}
}

func testComputerUseExecutionClient(t *testing.T) func(context.Context, string, map[string]any, string) (map[string]any, error) {
	t.Helper()
	imagePath := filepath.Join(t.TempDir(), "capture.png")
	if err := os.WriteFile(imagePath, testComputerUsePNG(t), 0o600); err != nil {
		t.Fatalf("write screenshot fixture: %v", err)
	}
	return func(_ context.Context, action string, _ map[string]any, _ string) (map[string]any, error) {
		switch action {
		case "SCREEN_CAPTURE":
			return map[string]any{
				"image_path":   imagePath,
				"image_width":  4,
				"image_height": 4,
				"display_id":   1,
				"scale_x":      1,
				"scale_y":      1,
				"origin_x":     0,
				"origin_y":     0,
			}, nil
		case "ACTIVE_WINDOW_INFO":
			return map[string]any{
				"title": "Login",
				"class": "Login",
			}, nil
		default:
			return map[string]any{"ok": true}, nil
		}
	}
}

func testComputerUsePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.SetRGBA(x, y, color.RGBA{G: 40, A: 255})
		}
	}
	buf := bytes.Buffer{}
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}
