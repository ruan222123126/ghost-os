package sessionturn

import "ghost-os/bridge/llm"

type ProjectionOptions struct {
	IdleTurns           int
	MicrocompactEnabled bool
	TraceID             string
}

func ProjectMessagesForModel(messages []llm.Message, options ProjectionOptions) []llm.Message {
	if len(messages) == 0 {
		return nil
	}
	if !options.MicrocompactEnabled {
		return llm.CloneMessages(messages)
	}
	return newMicrocompactProjector(options).Project(messages)
}
