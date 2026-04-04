package guiagent

import (
	"encoding/json"
	"fmt"
	"strings"
)

func ParseDecision(raw string) (Decision, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return Decision{}, &RunError{Code: ErrorModelOutputParse, Message: "model output is empty"}
	}

	var decision Decision
	decoder := json.NewDecoder(strings.NewReader(trimmed))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decision); err != nil {
		return Decision{}, &RunError{Code: ErrorModelOutputParse, Message: err.Error()}
	}
	if decoder.More() {
		return Decision{}, &RunError{Code: ErrorModelOutputParse, Message: "model output must contain exactly one JSON object"}
	}
	if err := validateDecision(decision); err != nil {
		return Decision{}, err
	}
	return decision, nil
}

type actionValidator func(Action) error

var decisionValidators = map[string]actionValidator{
	ActionClick:       requireBoxActionValidator(ActionClick),
	ActionDoubleClick: requireBoxActionValidator(ActionDoubleClick),
	ActionRightClick:  requireBoxActionValidator(ActionRightClick),
	ActionTypeText:    validateTypeAction,
	ActionHotkey:      validateHotkeyAction,
	ActionScroll:      validateScrollAction,
	ActionDrag:        validateDragAction,
	ActionWait:        validateWaitAction,
	ActionFinished:    validateFinishedAction,
	ActionCallUser:    validateCallUserAction,
}

func validateDecision(decision Decision) error {
	actionType := strings.TrimSpace(decision.Action.Type)
	if actionType == "" {
		return &RunError{Code: ErrorModelOutputParse, Message: "action.type is required"}
	}
	validator, ok := decisionValidators[actionType]
	if !ok {
		return &RunError{Code: ErrorModelOutputParse, Message: fmt.Sprintf("unsupported action type %q", actionType)}
	}
	return validator(decision.Action)
}

func requireBoxActionValidator(actionType string) actionValidator {
	return func(action Action) error {
		return requireTargetBox(action.Target, actionType)
	}
}

func validateTypeAction(action Action) error {
	if strings.TrimSpace(action.Text) == "" {
		return &RunError{Code: ErrorModelOutputParse, Message: "type action requires text"}
	}
	return nil
}

func validateHotkeyAction(action Action) error {
	if len(action.Keys) == 0 {
		return &RunError{Code: ErrorModelOutputParse, Message: "hotkey action requires keys"}
	}
	return nil
}

func validateScrollAction(action Action) error {
	if action.DeltaX == 0 && action.DeltaY == 0 {
		return &RunError{Code: ErrorModelOutputParse, Message: "scroll action requires delta_x or delta_y"}
	}
	return validateTarget(action.Target)
}

func validateDragAction(action Action) error {
	if err := requireTargetBox(action.Target, ActionDrag); err != nil {
		return err
	}
	return requireDestinationBox(action.Destination)
}

func validateWaitAction(action Action) error {
	if action.DurationMs < 0 {
		return &RunError{Code: ErrorModelOutputParse, Message: "wait duration_ms must be non-negative"}
	}
	return nil
}

func validateFinishedAction(Action) error {
	return nil
}

func validateCallUserAction(action Action) error {
	if strings.TrimSpace(action.Prompt) == "" {
		return &RunError{Code: ErrorModelOutputParse, Message: "call_user action requires prompt"}
	}
	return nil
}

func requireTargetBox(target *ActionTarget, actionType string) error {
	if err := validateTarget(target); err != nil {
		return err
	}
	if target == nil || target.Box == nil {
		return &RunError{Code: ErrorModelOutputParse, Message: fmt.Sprintf("%s action requires target.box", actionType)}
	}
	return nil
}

func requireDestinationBox(target *ActionTarget) error {
	if err := validateTarget(target); err != nil {
		return err
	}
	if target == nil || target.Box == nil {
		return &RunError{Code: ErrorModelOutputParse, Message: "drag action requires destination.box"}
	}
	return nil
}

func validateTarget(target *ActionTarget) error {
	if target == nil || target.Box == nil {
		return nil
	}
	for _, value := range target.Box[:] {
		if value < 0 || value > 1 {
			return &RunError{Code: ErrorInvalidTargetBox, Message: "target.box values must be between 0 and 1"}
		}
	}
	x1, y1, x2, y2 := target.Box[0], target.Box[1], target.Box[2], target.Box[3]
	if x2 <= x1 || y2 <= y1 {
		return &RunError{Code: ErrorInvalidTargetBox, Message: "target.box must satisfy x2>x1 and y2>y1"}
	}
	return nil
}
