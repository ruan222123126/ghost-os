package session

func clonePendingQuestions(raw map[string]PendingHumanQuestion) map[string]PendingHumanQuestion {
	if len(raw) == 0 {
		return nil
	}

	out := make(map[string]PendingHumanQuestion, len(raw))
	for id, question := range raw {
		question.Options = cloneHumanQuestionOptions(question.Options)
		out[id] = question
	}
	return out
}

func cloneHumanAnswers(raw map[string]string) map[string]string {
	if len(raw) == 0 {
		return nil
	}

	out := make(map[string]string, len(raw))
	for questionID, answer := range raw {
		out[questionID] = answer
	}
	return out
}

func cloneDynamicToolLoads(raw map[string]DynamicToolLoad) map[string]DynamicToolLoad {
	if len(raw) == 0 {
		return nil
	}

	out := make(map[string]DynamicToolLoad, len(raw))
	for toolName, load := range raw {
		out[toolName] = normalizeDynamicToolLoad(toolName, load)
	}
	return out
}

func cloneDynamicSkillLoads(raw map[string]DynamicSkillLoad) map[string]DynamicSkillLoad {
	if len(raw) == 0 {
		return nil
	}

	out := make(map[string]DynamicSkillLoad, len(raw))
	for skillName, load := range raw {
		out[skillName] = normalizeDynamicSkillLoad(skillName, load)
	}
	return out
}
