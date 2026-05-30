package orchestration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	bridgeTasks "ghost-os/bridge/tasks"
)

type LegacyOrchestrationMigrationReport struct {
	TasksPath        string
	MigratedTaskIDs  []string
	UnchangedTaskIDs []string
}

type legacyOrchestrationTaskFile struct {
	id   string
	path string
	task ScheduledTask
}

// DetectLegacyOrchestrationTasks 返回仍包含 legacy start/end 节点的 orchestration 任务 ID。
func DetectLegacyOrchestrationTasks(baseDir string) ([]string, error) {
	legacyTasks, tasksDir, err := scanLegacyOrchestrationTasks(baseDir)
	if err != nil {
		return nil, err
	}
	_ = tasksDir
	ids := make([]string, 0, len(legacyTasks))
	for _, taskFile := range legacyTasks {
		ids = append(ids, taskFile.id)
	}
	return ids, nil
}

// MigrateLegacyOrchestrations 显式移除 orchestration 任务中的 legacy start/end 边界节点并写回。
func MigrateLegacyOrchestrations(baseDir string) (LegacyOrchestrationMigrationReport, error) {
	legacyTasks, tasksDir, err := scanLegacyOrchestrationTasks(baseDir)
	if err != nil {
		return LegacyOrchestrationMigrationReport{}, err
	}
	report := LegacyOrchestrationMigrationReport{
		TasksPath:       tasksDir,
		MigratedTaskIDs: make([]string, 0, len(legacyTasks)),
	}
	for _, taskFile := range legacyTasks {
		migrated := taskFile.task
		migrated.Orchestration = stripLegacyBoundaryNodes(migrated.Orchestration)
		if err := validateTaskDefinition(&migrated); err != nil {
			return LegacyOrchestrationMigrationReport{}, fmt.Errorf("migrate orchestration task %q: %w", taskFile.id, err)
		}
		if err := writeMigratedTaskFile(taskFile.path, migrated); err != nil {
			return LegacyOrchestrationMigrationReport{}, fmt.Errorf("write migrated orchestration task %q: %w", taskFile.id, err)
		}
		report.MigratedTaskIDs = append(report.MigratedTaskIDs, taskFile.id)
	}
	return report, nil
}

func scanLegacyOrchestrationTasks(baseDir string) ([]legacyOrchestrationTaskFile, string, error) {
	store, err := bridgeTasks.NewStore(baseDir, nil)
	if err != nil {
		return nil, "", err
	}
	tasksDir := strings.TrimSpace(store.TasksDir())
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		return nil, tasksDir, fmt.Errorf("scan tasks directory %s: %w", tasksDir, err)
	}

	legacyTasks := make([]legacyOrchestrationTaskFile, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		taskID := strings.TrimSpace(strings.TrimSuffix(entry.Name(), ".json"))
		if !bridgeTasks.IsValidTaskID(taskID) {
			continue
		}
		path := filepath.Join(tasksDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, tasksDir, fmt.Errorf("read orchestration task file %s: %w", path, err)
		}
		var task ScheduledTask
		if err := json.Unmarshal(data, &task); err != nil {
			return nil, tasksDir, fmt.Errorf("decode orchestration task file %s: %w", path, err)
		}
		if strings.TrimSpace(task.ID) == "" {
			task.ID = taskID
		}
		if normalizeTaskKind(task.TaskKind) != taskKindOrchestration || !hasLegacyBoundaryNodes(task.Orchestration) {
			continue
		}
		legacyTasks = append(legacyTasks, legacyOrchestrationTaskFile{
			id:   taskID,
			path: path,
			task: task,
		})
	}
	return legacyTasks, tasksDir, nil
}

func hasLegacyBoundaryNodes(definition *OrchestrationDefinition) bool {
	if definition == nil {
		return false
	}
	for _, node := range definition.Nodes {
		if node.Type == orchestrationNodeTypeStart || node.Type == orchestrationNodeTypeEnd {
			return true
		}
	}
	return false
}

func stripLegacyBoundaryNodes(definition *OrchestrationDefinition) *OrchestrationDefinition {
	if definition == nil {
		return nil
	}
	next := cloneTaskOrchestration(definition)
	boundaryIDs := make(map[string]struct{})
	nodes := make([]OrchestrationNode, 0, len(next.Nodes))
	for _, node := range next.Nodes {
		if node.Type == orchestrationNodeTypeStart || node.Type == orchestrationNodeTypeEnd {
			boundaryIDs[node.ID] = struct{}{}
			continue
		}
		nodes = append(nodes, node)
	}
	edges := make([]OrchestrationEdge, 0, len(next.Edges))
	for _, edge := range next.Edges {
		if _, blocked := boundaryIDs[edge.FromNodeID]; blocked {
			continue
		}
		if _, blocked := boundaryIDs[edge.ToNodeID]; blocked {
			continue
		}
		edges = append(edges, edge)
	}
	next.Nodes = nodes
	next.Edges = edges
	return next
}

func writeMigratedTaskFile(path string, task ScheduledTask) error {
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
