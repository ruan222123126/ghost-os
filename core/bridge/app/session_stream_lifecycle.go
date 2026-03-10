package app

import "ghost-os/bridge/agent"

func newSessionStreamLifecyclePayloadBuilder(turn *sessionTurnState) agent.StreamLifecyclePayloadBuilder {
	return agent.StreamLifecyclePayloadBuilder{
		SessionID: func() string {
			return turn.currentSessionID()
		},
		BuildRunStarted: func() (any, error) {
			return map[string]any{
				"session_id": turn.currentSessionID(),
			}, nil
		},
		BuildMessage: func(response string) (any, error) {
			normalized, _, err := parseSessionEndSignal(response)
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"text":       normalized,
				"session_id": turn.currentSessionID(),
			}, nil
		},
		BuildDone: func(response string) (any, error) {
			_, sessionEnd, err := parseSessionEndSignal(response)
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"session_id":    turn.currentSessionID(),
				"session_ended": sessionEnd != nil,
			}, nil
		},
	}
}
