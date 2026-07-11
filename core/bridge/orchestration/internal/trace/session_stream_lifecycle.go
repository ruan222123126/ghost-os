package trace

import "ghost-os/bridge/agent"

type ParseSessionEndFunc func(response string) (normalized string, sessionEnded bool, err error)

func NewSessionStreamLifecyclePayloadBuilder(
	sessionID func() string,
	parseSessionEnd ParseSessionEndFunc,
) agent.StreamLifecyclePayloadBuilder {
	return agent.StreamLifecyclePayloadBuilder{
		SessionID: func() string {
			return sessionID()
		},
		BuildRunStarted: func() (any, error) {
			return map[string]any{
				"session_id": sessionID(),
			}, nil
		},
		BuildMessage: func(response string) (any, error) {
			normalized, _, err := parseSessionEnd(response)
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"text":       normalized,
				"session_id": sessionID(),
			}, nil
		},
		BuildDone: func(response string) (any, error) {
			_, sessionEnded, err := parseSessionEnd(response)
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"session_id":    sessionID(),
				"session_ended": sessionEnded,
			}, nil
		},
	}
}
