package migration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ghost-os/bridge/taskdefs"
	bridgeTasks "ghost-os/bridge/tasks"
)

const (
	legacyNodeTypeStart = "start"
	legacyNodeTypeEnd   = "end"
)

type Report struct {
	TasksPath        string
	MigratedTaskIDs  []string
	UnchangedTaskIDs []string
}

type TaskValidator func(*taskdefs.ScheduledTask) error

type taskFile struct {
	id   string
	path string
	task taskdefs.ScheduledTask
}

func Detect(baseDir string) ([]string, error) {
	legacyTasks, _, err := scan(baseDir)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(legacyTasks))
	for _, taskFile := range legacyTasks {
		ids = append(ids, taskFile.id)
	}
	return ids, nil
}

func Migrate(baseDir string, validate TaskValidator) (Report, error) {
	legacyTasks, tasksDir, err := scan(baseDir)
	if err != nil {
		return Report{}, err
	}
	report := Report{
		TasksPath:       tasksDir,
		MigratedTaskIDs: make([]string, 0, len(legacyTasks)),
	}
	for _, taskFile := range legacyTasks {
		migrated := taskFile.task
		migrated.Orchestration = stripLegacyBoundaryNodes(migrated.Orchestration)
		if validate != nil {
			if err := validate(&migrated); err != nil {
				return Report{}, fmt.Errorf("migrate orchestration task %q: %w", taskFile.id, err)
			}
		}
		if err := writeTaskFile(taskFile.path, migrated); err != nil {
			return Report{}, fmt.Errorf("write migrated orchestration task %q: %w", taskFile.id, err)
		}
		report.MigratedTaskIDs = append(report.MigratedTaskIDs, taskFile.id)
	}
	return report, nil
}

func scan(baseDir string) ([]taskFile, string, error) {
	store, err := bridgeTasks.NewStore(baseDir, nil)
	if err != nil {
		return nil, "", err
	}
	tasksDir := strings.TrimSpace(store.TasksDir())
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		return nil, tasksDir, fmt.Errorf("scan tasks directory %s: %w", tasksDir, err)
	}

	legacyTasks := make([]taskFile, 0)
	for _, entry := range entries {
		taskFile, ok, err := loadLegacyTaskFile(tasksDir, entry)
		if err != nil {
			return nil, tasksDir, err
		}
		if ok {
			legacyTasks = append(legacyTasks, taskFile)
		}
	}
	return legacyTasks, tasksDir, nil
}

func loadLegacyTaskFile(tasksDir string, entry os.DirEntry) (taskFile, bool, error) {
	if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
		return taskFile{}, false, nil
	}
	taskID := strings.TrimSpace(strings.TrimSuffix(entry.Name(), ".json"))
	if !bridgeTasks.IsValidTaskID(taskID) {
		return taskFile{}, false, nil
	}
	path := filepath.Join(tasksDir, entry.Name())
	task, err := readTaskFile(path)
	if err != nil {
		return taskFile{}, false, err
	}
	if strings.TrimSpace(task.ID) == "" {
		task.ID = taskID
	}
	if taskdefs.NormalizeTaskKind(task.TaskKind) != taskdefs.KindOrchestration ||
		!hasLegacyBoundaryNodes(task.Orchestration) {
		return taskFile{}, false, nil
	}
	return taskFile{id: taskID, path: path, task: task}, true, nil
}

func readTaskFile(path string) (taskdefs.ScheduledTask, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return taskdefs.ScheduledTask{}, fmt.Errorf("read orchestration task file %s: %w", path, err)
	}
	var task taskdefs.ScheduledTask
	if err := json.Unmarshal(data, &task); err != nil {
		return taskdefs.ScheduledTask{}, fmt.Errorf("decode orchestration task file %s: %w", path, err)
	}
	return task, nil
}

func hasLegacyBoundaryNodes(definition *taskdefs.OrchestrationDefinition) bool {
	if definition == nil {
		return false
	}
	for _, node := range definition.Nodes {
		if isLegacyBoundaryNode(node.Type) {
			return true
		}
	}
	return false
}

func stripLegacyBoundaryNodes(
	definition *taskdefs.OrchestrationDefinition,
) *taskdefs.OrchestrationDefinition {
	if definition == nil {
		return nil
	}
	next := taskdefs.CloneOrchestrationDefinition(definition)
	boundaryIDs := make(map[string]struct{})
	nodes := make([]taskdefs.OrchestrationNode, 0, len(next.Nodes))
	for _, node := range next.Nodes {
		if isLegacyBoundaryNode(node.Type) {
			boundaryIDs[node.ID] = struct{}{}
			continue
		}
		nodes = append(nodes, node)
	}
	next.Nodes = nodes
	next.Edges = stripBoundaryEdges(next.Edges, boundaryIDs)
	return next
}

func stripBoundaryEdges(
	edges []taskdefs.OrchestrationEdge,
	boundaryIDs map[string]struct{},
) []taskdefs.OrchestrationEdge {
	next := make([]taskdefs.OrchestrationEdge, 0, len(edges))
	for _, edge := range edges {
		if _, blocked := boundaryIDs[edge.FromNodeID]; blocked {
			continue
		}
		if _, blocked := boundaryIDs[edge.ToNodeID]; blocked {
			continue
		}
		next = append(next, edge)
	}
	return next
}

func isLegacyBoundaryNode(nodeType string) bool {
	switch strings.TrimSpace(nodeType) {
	case legacyNodeTypeStart, legacyNodeTypeEnd:
		return true
	default:
		return false
	}
}

func writeTaskFile(path string, task taskdefs.ScheduledTask) error {
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal task %q: %w", task.ID, err)
	}
	data = append(data, '\n')
	tempPath := fmt.Sprintf("%s.tmp-%d", path, time.Now().UTC().UnixNano())
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return fmt.Errorf("write temp task file %s: %w", tempPath, err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("replace task file %s: %w", path, err)
	}
	return nil
}
