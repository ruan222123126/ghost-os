package orchestration

import "ghost-os/bridge/llm"

type messageProjectionOptions struct {
	IdleTurns           int
	MicrocompactEnabled bool
	TraceID             string
}

func projectMessagesForModel(messages []llm.Message, options messageProjectionOptions) []llm.Message {
	if len(messages) == 0 {
		return nil
	}
	if !options.MicrocompactEnabled {
		return llm.CloneMessages(messages)
	}
	return newMicrocompactProjector(options).Project(messages)
}
