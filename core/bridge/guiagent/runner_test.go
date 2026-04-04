package guiagent

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"testing"

	"ghost-os/bridge/artifacts"
	"ghost-os/bridge/llm"
)

type runnerTestCompleter struct {
	outputs []string
	index   int
}

func (c *runnerTestCompleter) Complete(
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

type runnerTestOperator struct {
	observations []Observation
	actions      []Action
}

func (o *runnerTestOperator) Observe(
	_ context.Context,
	_ Request,
	_ string,
) (Observation, error) {
	observation := o.observations[0]
	o.observations = o.observations[1:]
	return observation, nil
}

func (o *runnerTestOperator) Execute(
	_ context.Context,
	_ Request,
	_ Observation,
	action Action,
	_ string,
) (map[string]any, error) {
	o.actions = append(o.actions, action)
	return map[string]any{"ok": true}, nil
}

func TestRunnerCompletesAfterVisibleStep(t *testing.T) {
	store, err := artifacts.NewSessionArtifactStore(t.TempDir())
	if err != nil {
		t.Fatalf("new artifact store: %v", err)
	}
	writer, err := NewArtifactWriter(store, "sess-runner", "run-1")
	if err != nil {
		t.Fatalf("new writer: %v", err)
	}
	operator := &runnerTestOperator{
		observations: []Observation{
			testObservation(t, color.RGBA{R: 10, A: 255}),
			testObservation(t, color.RGBA{R: 200, A: 255}),
			testObservation(t, color.RGBA{R: 200, A: 255}),
		},
	}
	runner := NewRunner(&runnerTestCompleter{
		outputs: []string{
			`{"thought":"click login","action":{"type":"click","target":{"box":[0.1,0.1,0.2,0.2]}}}`,
			`{"thought":"done","action":{"type":"finished"}}`,
		},
	}, operator, writer, 4)

	result, err := runner.Continue(context.Background(), State{
		RunID: "run-1",
		Request: Request{
			Goal: "open settings",
			Mode: "desktop",
		},
	}, "", "trace-runner")
	if err != nil {
		t.Fatalf("runner failed: %v", err)
	}
	if result.Status != StatusSuccess || !result.Completed {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(operator.actions) != 1 || operator.actions[0].Type != ActionClick {
		t.Fatalf("unexpected actions: %+v", operator.actions)
	}
	if len(result.StepSummaries) != 1 || !result.StepSummaries[0].VisibleEffect {
		t.Fatalf("unexpected step summaries: %+v", result.StepSummaries)
	}
}

func TestRunnerReturnsAwaitingUserAndResumes(t *testing.T) {
	store, err := artifacts.NewSessionArtifactStore(t.TempDir())
	if err != nil {
		t.Fatalf("new artifact store: %v", err)
	}
	writer, err := NewArtifactWriter(store, "sess-runner", "run-2")
	if err != nil {
		t.Fatalf("new writer: %v", err)
	}
	runner := NewRunner(&runnerTestCompleter{
		outputs: []string{
			`{"thought":"need otp","action":{"type":"call_user","prompt":"Please enter the OTP"}}`,
			`{"thought":"done","action":{"type":"finished"}}`,
		},
	}, &runnerTestOperator{
		observations: []Observation{
			testObservation(t, color.RGBA{B: 30, A: 255}),
			testObservation(t, color.RGBA{B: 30, A: 255}),
		},
	}, writer, 4)

	initial, err := runner.Continue(context.Background(), State{
		RunID: "run-2",
		Request: Request{
			Goal: "finish login",
			Mode: "desktop",
		},
	}, "", "trace-call-user")
	if err != nil {
		t.Fatalf("initial run failed: %v", err)
	}
	if initial.Status != StatusAwaitingUser || initial.AwaitingUser == nil {
		t.Fatalf("unexpected initial result: %+v", initial)
	}

	resumed, err := runner.Continue(context.Background(), *initial.State, "123456", "trace-resume")
	if err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	if resumed.Status != StatusSuccess || !resumed.Completed {
		t.Fatalf("unexpected resumed result: %+v", resumed)
	}
	if resumed.State == nil || resumed.State.Steps[0].UserAnswer != "123456" {
		t.Fatalf("expected user answer to be recorded: %+v", resumed.State)
	}
}

func testObservation(t *testing.T, fill color.RGBA) Observation {
	t.Helper()
	buf := bytes.Buffer{}
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.SetRGBA(x, y, fill)
		}
	}
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return Observation{
		ImageWidth:        4,
		ImageHeight:       4,
		DisplayID:         1,
		ScaleX:            1,
		ScaleY:            1,
		OriginX:           0,
		OriginY:           0,
		ActiveWindowTitle: "Settings",
		ActiveWindowClass: "Settings",
		ImageBytes:        buf.Bytes(),
	}
}
