package task

import (
	"strings"

	bridgeTasks "ghost-os/bridge/tasks"
)

func CollectRunTranscriptSessionIDs(result bridgeTasks.ExecutionResult) []string {
	collector := sessionIDCollector{seen: map[string]struct{}{}}
	for _, node := range result.NodeResults {
		collector.collectNode(node)
	}
	return collector.ids
}

type sessionIDCollector struct {
	ids  []string
	seen map[string]struct{}
}

func (c *sessionIDCollector) collectNode(node bridgeTasks.RunNodeResult) {
	if node.NodeType == workflowNodeAgent {
		c.add(transcriptString(transcriptRecord(node.Output), "session_id_output"))
		return
	}
	output := transcriptRecord(node.Output)
	c.collectMembers(transcriptSlice(output, "member_results"))
	for _, item := range transcriptSlice(output, "dispatch_results") {
		c.collectMembers(transcriptSlice(transcriptRecord(item), "member_results"))
	}
}

func (c *sessionIDCollector) collectMembers(items []any) {
	for _, item := range items {
		c.add(transcriptString(transcriptRecord(item), "session_id"))
	}
}

func (c *sessionIDCollector) add(sessionID string) {
	trimmed := strings.TrimSpace(sessionID)
	if trimmed == "" {
		return
	}
	if _, exists := c.seen[trimmed]; exists {
		return
	}
	c.seen[trimmed] = struct{}{}
	c.ids = append(c.ids, trimmed)
}
