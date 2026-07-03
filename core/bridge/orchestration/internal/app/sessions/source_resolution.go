package sessions

import (
	"strings"

	"ghost-os/bridge/internal/stringutil"
	"ghost-os/bridge/orchestration/internal/contracts/api"
	bridgeTasks "ghost-os/bridge/tasks"
)

const (
	SessionSourceKindWorkflow      = "workflow"
	SessionSourceKindOrchestration = "orchestration"
	SessionSourceKindLoop          = "loop"
	SessionSourceKindTask          = "task"
)

type SessionSourceAssignment = api.SessionSourceAssignment
type SessionSourceResolution = api.SessionSourceResolution

type SessionSourceTask struct {
	ID, Name, Message, TaskKind string
	AgentMode                   string
	RunLogs                     []bridgeTasks.RunLog
}

var sessionSourcePriority = map[string]int{
	SessionSourceKindWorkflow:      4,
	SessionSourceKindOrchestration: 3,
	SessionSourceKindLoop:          2,
	SessionSourceKindTask:          1,
}

func BuildSessionSourceResolution(tasks []SessionSourceTask) (SessionSourceResolution, error) {
	index := buildSessionSourceTaskIndex(tasks)
	resolution := SessionSourceResolution{
		Assignments:      map[string]SessionSourceAssignment{},
		HiddenSessionIDs: []string{},
	}
	hiddenSessionIDs := map[string]struct{}{}

	for _, task := range tasks {
		for _, run := range task.RunLogs {
			source := sessionSourceFromRun(run, index)
			if source.Kind == "" {
				continue
			}
			if !isVisibleSessionSourceRun(run) {
				for _, sessionID := range runSessionIDs(run, source.Kind) {
					addSessionSourceID(hiddenSessionIDs, sessionID)
				}
				continue
			}
			assignRunOutputSession(&resolution, run.SessionIDOutput, source)
			for _, sessionID := range hiddenSessionIDsFromRun(run, source.Kind) {
				addSessionSourceID(hiddenSessionIDs, sessionID)
			}
		}
	}

	resolution.HiddenSessionIDs = unresolvedHiddenSessionIDs(hiddenSessionIDs, resolution.Assignments)
	return resolution, nil
}

func BuildSessionSourceTask(
	task bridgeTasks.ScheduledTask,
	runLogs []bridgeTasks.RunLog,
) SessionSourceTask {
	return SessionSourceTask{
		ID:        strings.TrimSpace(task.ID),
		Name:      strings.TrimSpace(task.Name),
		Message:   strings.TrimSpace(task.Message),
		TaskKind:  bridgeTasks.NormalizeKind(task.TaskKind),
		AgentMode: bridgeTasks.NormalizeAgentMode(task.AgentMode),
		RunLogs:   append([]bridgeTasks.RunLog(nil), runLogs...),
	}
}

type sessionSourceTaskIndex struct {
	names, kinds map[string]string
	loopTaskIDs  map[string]struct{}
}

type resolvedSessionSource struct {
	Kind, OwnerID string
	OwnerName     string
}

func buildSessionSourceTaskIndex(tasks []SessionSourceTask) sessionSourceTaskIndex {
	index := sessionSourceTaskIndex{
		names:       make(map[string]string, len(tasks)),
		kinds:       make(map[string]string, len(tasks)),
		loopTaskIDs: map[string]struct{}{},
	}
	for _, task := range tasks {
		id := strings.TrimSpace(task.ID)
		if id == "" {
			continue
		}
		index.names[id] = sourceNameFromTask(task)
		index.kinds[id] = bridgeTasks.NormalizeKind(task.TaskKind)
		if isLoopScheduledTask(task) {
			index.loopTaskIDs[id] = struct{}{}
		}
	}
	return index
}

func sourceNameFromTask(task SessionSourceTask) string {
	id := strings.TrimSpace(task.ID)
	if bridgeTasks.NormalizeKind(task.TaskKind) == bridgeTasks.KindOrchestration {
		return stringutil.FirstNonEmpty(task.Name, id)
	}
	if bridgeTasks.NormalizeKind(task.TaskKind) == bridgeTasks.KindAgentMessage {
		return stringutil.FirstNonEmpty(previewTaskMessage(task.Message), id)
	}
	return id
}

func previewTaskMessage(message string) string {
	const taskNamePreviewLength = 48
	firstLine := strings.TrimSpace(strings.Split(strings.TrimSpace(message), "\n")[0])
	runes := []rune(firstLine)
	if len(runes) <= taskNamePreviewLength {
		return firstLine
	}
	return string(runes[:taskNamePreviewLength]) + "..."
}

func sessionSourceFromRun(run bridgeTasks.RunLog, index sessionSourceTaskIndex) resolvedSessionSource {
	kind := sourceKindFromRun(run, index)
	if kind == "" {
		return resolvedSessionSource{}
	}
	ownerID := strings.TrimSpace(run.TaskID)
	if ownerID == "" {
		return resolvedSessionSource{}
	}
	return resolvedSessionSource{
		Kind:      kind,
		OwnerID:   ownerID,
		OwnerName: stringutil.FirstNonEmpty(index.names[ownerID], ownerID),
	}
}

func sourceKindFromRun(run bridgeTasks.RunLog, index sessionSourceTaskIndex) string {
	taskID := strings.TrimSpace(run.TaskID)
	runKind := bridgeTasks.NormalizeKind(stringutil.FirstNonEmpty(run.TaskKind, index.kinds[taskID]))
	if runKind == bridgeTasks.KindWorkflow {
		return SessionSourceKindWorkflow
	}
	if runKind == bridgeTasks.KindOrchestration {
		return SessionSourceKindOrchestration
	}
	if runKind == bridgeTasks.KindAgentMessage && strings.TrimSpace(run.SessionIDInput) == "" {
		if _, ok := index.loopTaskIDs[taskID]; ok {
			return SessionSourceKindLoop
		}
		return SessionSourceKindTask
	}
	return ""
}

func assignRunOutputSession(
	resolution *SessionSourceResolution,
	sessionID string,
	source resolvedSessionSource,
) {
	id := strings.TrimSpace(sessionID)
	if id == "" {
		return
	}
	existing, ok := resolution.Assignments[id]
	if ok && sessionSourcePriority[existing.Kind] >= sessionSourcePriority[source.Kind] {
		return
	}
	resolution.Assignments[id] = SessionSourceAssignment{
		Kind:      source.Kind,
		OwnerID:   source.OwnerID,
		OwnerName: source.OwnerName,
	}
}

func isVisibleSessionSourceRun(run bridgeTasks.RunLog) bool {
	switch strings.TrimSpace(run.Status) {
	case bridgeTasks.RunStatusSuccess,
		bridgeTasks.RunStatusIncomplete,
		bridgeTasks.RunStatusCancelled,
		bridgeTasks.RunStatusError,
		bridgeTasks.RunStatusSkipped:
		return true
	default:
		return false
	}
}

func runSessionIDs(run bridgeTasks.RunLog, sourceKind string) []string {
	ids := map[string]struct{}{}
	addSessionSourceID(ids, run.SessionIDOutput)
	for _, sessionID := range hiddenSessionIDsFromRun(run, sourceKind) {
		addSessionSourceID(ids, sessionID)
	}
	return mapKeys(ids)
}

func hiddenSessionIDsFromRun(run bridgeTasks.RunLog, sourceKind string) []string {
	if sourceKind == SessionSourceKindWorkflow {
		return workflowExecutionSessionIDs(run)
	}
	if sourceKind == SessionSourceKindOrchestration {
		return orchestrationExecutionSessionIDs(run)
	}
	return nil
}

func workflowExecutionSessionIDs(run bridgeTasks.RunLog) []string {
	ids := map[string]struct{}{}
	for _, node := range run.NodeResults {
		addSessionSourceID(ids, stringFromRecord(recordFromAny(node.Output), "session_id_output"))
	}
	return mapKeys(ids)
}

func orchestrationExecutionSessionIDs(run bridgeTasks.RunLog) []string {
	ids := map[string]struct{}{}
	for _, node := range run.NodeResults {
		addOrchestrationOutputSessionIDs(ids, recordFromAny(node.Output))
	}
	return mapKeys(ids)
}

func addOrchestrationOutputSessionIDs(ids map[string]struct{}, output map[string]any) {
	if len(output) == 0 {
		return
	}
	addSessionSourceID(ids, stringFromRecord(output, "owner_session_id"))
	addRecordStringValues(ids, recordFromAny(output["member_session_ids"]))
	addMemberResultSessionIDs(ids, arrayFromAny(output["member_results"]))
	for _, dispatch := range arrayFromAny(output["dispatch_results"]) {
		addMemberResultSessionIDs(ids, arrayFromAny(recordFromAny(dispatch)["member_results"]))
	}
}

func addMemberResultSessionIDs(ids map[string]struct{}, items []any) {
	for _, item := range items {
		addSessionSourceID(ids, stringFromRecord(recordFromAny(item), "session_id"))
	}
}

func addRecordStringValues(ids map[string]struct{}, record map[string]any) {
	for _, value := range record {
		addSessionSourceID(ids, stringFromAny(value))
	}
}

func unresolvedHiddenSessionIDs(
	hiddenSessionIDs map[string]struct{},
	assignments map[string]SessionSourceAssignment,
) []string {
	ids := make([]string, 0, len(hiddenSessionIDs))
	for sessionID := range hiddenSessionIDs {
		if _, ok := assignments[sessionID]; !ok {
			ids = append(ids, sessionID)
		}
	}
	return ids
}

func isLoopScheduledTask(task SessionSourceTask) bool {
	return bridgeTasks.NormalizeKind(task.TaskKind) == bridgeTasks.KindAgentMessage &&
		bridgeTasks.NormalizeAgentMode(task.AgentMode) == bridgeTasks.AgentModeRelay
}
