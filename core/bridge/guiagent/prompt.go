package guiagent

import (
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
)

const maxPromptHistorySteps = 5

const systemPrompt = `You are a desktop GUI executor.
Return exactly one JSON object with fields "thought" and "action".
Do not return markdown fences or extra prose.
Inside "action", use field "type" to choose the action name.
Strict shape example: {"thought":"done","action":{"type":"finished"}}.
Never use {"action":{"action":"..."}}.
Use normalized coordinates in target.box and destination.box as [x1,y1,x2,y2] within 0..1.
Available actions: click, double_click, right_click, type, hotkey, scroll, drag, wait, finished, call_user.
For click actions provide action.target.box.
For drag provide action.target.box and action.destination.box.
For type provide action.text.
For hotkey provide action.keys as an array of key names.
For scroll provide action.delta_x and/or action.delta_y. Positive delta_y scrolls down.
For wait provide action.duration_ms.
For call_user provide action.prompt and optional action.reason.
Use call_user when the task is blocked on user interaction such as login, captcha, or permission approval.
If the previous step had no visible effect, treat it as a failed attempt and choose a different action.`

func BuildMessages(request Request, state State, observation Observation) []llm.Message {
	userText := buildUserPrompt(request, state, observation)
	userContent := []llm.ContentPart{{
		Type: llm.ContentTypeText,
		Text: userText,
	}}
	if observation.Artifact != nil && strings.TrimSpace(observation.Artifact.Path) != "" {
		userContent = append(userContent, llm.ContentPart{
			Type: llm.ContentTypeImage,
			Image: &llm.ImageContent{
				Path:     observation.Artifact.Path,
				MimeType: observation.Artifact.MimeType,
				Width:    observation.Artifact.Width,
				Height:   observation.Artifact.Height,
				SHA256:   observation.Artifact.SHA256,
				Bytes:    observation.Artifact.Bytes,
			},
		})
	}
	return []llm.Message{
		{Role: llm.RoleSystem, Text: systemPrompt},
		{Role: llm.RoleUser, Content: userContent},
	}
}

func buildUserPrompt(request Request, state State, observation Observation) string {
	lines := []string{
		fmt.Sprintf("Goal: %s", strings.TrimSpace(request.Goal)),
		fmt.Sprintf("Mode: %s", strings.TrimSpace(request.Mode)),
		formatWindowTarget(request.Target),
		formatObservation(observation),
	}
	history := formatRecentHistory(state.Steps)
	if history != "" {
		lines = append(lines, "Recent steps:", history)
	}
	lines = append(lines, "Return the next single action as strict JSON.")
	return strings.Join(filterPromptLines(lines), "\n")
}

func formatWindowTarget(target WindowTarget) string {
	title := strings.TrimSpace(target.WindowTitle)
	className := strings.TrimSpace(target.WindowClass)
	switch {
	case title != "" && className != "":
		return fmt.Sprintf("Target window: title contains %q and class contains %q", title, className)
	case title != "":
		return fmt.Sprintf("Target window: title contains %q", title)
	case className != "":
		return fmt.Sprintf("Target window: class contains %q", className)
	default:
		return ""
	}
}

func formatObservation(observation Observation) string {
	return fmt.Sprintf(
		"Observation: display=%d size=%dx%d origin=(%d,%d) scale=(%.2f,%.2f) active_window_title=%q active_window_class=%q",
		observation.DisplayID,
		observation.ImageWidth,
		observation.ImageHeight,
		observation.OriginX,
		observation.OriginY,
		observation.ScaleX,
		observation.ScaleY,
		observation.ActiveWindowTitle,
		observation.ActiveWindowClass,
	)
}

func formatRecentHistory(steps []StepRecord) string {
	if len(steps) == 0 {
		return ""
	}
	start := len(steps) - maxPromptHistorySteps
	if start < 0 {
		start = 0
	}
	lines := make([]string, 0, len(steps)-start)
	for _, step := range steps[start:] {
		lines = append(lines, formatStepForPrompt(step))
	}
	return strings.Join(lines, "\n")
}

func formatStepForPrompt(step StepRecord) string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("%d. action=%s", step.Index, step.Action.Type))
	if step.UserAnswer != "" {
		builder.WriteString(fmt.Sprintf(" user_answer=%q", step.UserAnswer))
	}
	if step.Verification.Message != "" {
		builder.WriteString(fmt.Sprintf(" verification=%q", step.Verification.Message))
	}
	builder.WriteString(fmt.Sprintf(" visible_effect=%t", step.Verification.VisibleEffect))
	return builder.String()
}

func filterPromptLines(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
