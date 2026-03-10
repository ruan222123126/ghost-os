package agent

type StreamLifecyclePayloadBuilder struct {
	BuildRunStarted func() (any, error)
	BuildMessage    func(response string) (any, error)
	BuildDone       func(response string) (any, error)
}

func (b StreamLifecyclePayloadBuilder) runStartedPayload() (any, error) {
	if b.BuildRunStarted == nil {
		return map[string]any{}, nil
	}
	return b.BuildRunStarted()
}

func (b StreamLifecyclePayloadBuilder) messagePayload(response string) (any, error) {
	if b.BuildMessage == nil {
		return map[string]any{"text": response}, nil
	}
	return b.BuildMessage(response)
}

func (b StreamLifecyclePayloadBuilder) donePayload(response string) (any, error) {
	if b.BuildDone == nil {
		return map[string]any{}, nil
	}
	return b.BuildDone(response)
}
