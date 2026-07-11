package llm

// ProjectMessagesForCompletion maps persisted internal-only roles to provider-safe roles
// without mutating the stored transcript.
func ProjectMessagesForCompletion(messages []Message) []Message {
	projected := CloneMessages(messages)
	for index := range projected {
		if projected[index].Role == RoleInternal {
			projected[index].Role = RoleAssistant
		}
	}
	return projected
}
