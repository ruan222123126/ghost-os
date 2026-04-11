package orchestration

import (
	"fmt"
	"strings"
)

func (r taskMutationRunner) ensureTaskSessionExists(taskKind string, sessionID string) error {
	if normalizeTaskKind(taskKind) != taskKindAgentMessage {
		return nil
	}
	id := strings.TrimSpace(sessionID)
	if id == "" || r.sessionStore == nil {
		return nil
	}
	sess, err := r.sessionStore.Load(id)
	if err != nil {
		return err
	}
	if sess.IsEnded() {
		return errSessionEnded
	}
	return nil
}

func ensureWorkflowAllowedForTaskKind(taskKind string, workflow *WorkflowDefinition) error {
	normalized := normalizeTaskKind(taskKind)
	if workflow == nil || normalized == taskKindWorkflow {
		return nil
	}
	if !isSupportedTaskKind(normalized) {
		return invalidTaskConfig(fmt.Sprintf("unsupported task_kind %q", normalized))
	}
	return invalidTaskConfig(normalized + " does not allow workflow")
}
