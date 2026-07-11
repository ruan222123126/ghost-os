package task

import (
	"fmt"
	"strings"

	"ghost-os/bridge/taskdefs"
)

func (b *runTranscriptBuilder) appendWorkflowResults() {
	if len(b.options.Result.NodeResults) == 0 {
		b.addEvent("工作流没有产生节点结果")
		return
	}
	for _, node := range b.options.Result.NodeResults {
		b.appendWorkflowNode(node)
	}
}

func (b *runTranscriptBuilder) appendWorkflowNode(node taskdefs.RunNodeResult) {
	if strings.TrimSpace(node.Error) != "" {
		b.addEvent(workflowNodeErrorEvent(node))
	}
	switch node.NodeType {
	case workflowNodeStart:
		b.addEvent(workflowStartEvent(node))
	case workflowNodeEnd:
		b.addEvent("工作流结束：" + node.NodeID)
	case workflowNodeIf:
		b.addEvent(workflowIfEvent(node))
	case workflowNodeLoop:
		b.addEvent(workflowLoopEvent(node))
	case workflowNodeTool:
		b.appendWorkflowToolNode(node)
	case workflowNodeLLM:
		b.appendWorkflowTextNode(node, "LLM")
	case workflowNodeAgent:
		b.appendWorkflowAgentNode(node)
	}
}

func workflowNodeErrorEvent(node taskdefs.RunNodeResult) string {
	return fmt.Sprintf("节点执行失败：%s (%s)\nerror: %s", node.NodeID, node.NodeType, strings.TrimSpace(node.Error))
}

func workflowStartEvent(node taskdefs.RunNodeResult) string {
	output := transcriptRecord(node.Output)
	nextIDs := transcriptStringSlice(output["next_node_ids"])
	if len(nextIDs) > 0 {
		return fmt.Sprintf("工作流并行分支：%s -> %s", node.NodeID, strings.Join(nextIDs, ", "))
	}
	nextID := transcriptString(output, "next_node_id")
	return fmt.Sprintf("工作流开始：%s -> %s", node.NodeID, nextID)
}

func workflowIfEvent(node taskdefs.RunNodeResult) string {
	output := transcriptNestedRecord(node.Output, "output")
	branch := transcriptString(output, "branch")
	nextID := transcriptString(output, "next_node_id")
	operator := transcriptString(output, "operator")
	value := transcriptString(output, "value")
	return fmt.Sprintf("工作流 if 判断：%s branch=%s operator=%s value=%s -> %s", node.NodeID, branch, operator, value, nextID)
}

func workflowLoopEvent(node taskdefs.RunNodeResult) string {
	output := transcriptNestedRecord(node.Output, "output")
	nextID := transcriptString(output, "next_node_id")
	if !transcriptBool(output, "entering_loop") {
		return fmt.Sprintf("工作流退出循环：%s -> %s", node.NodeID, nextID)
	}
	iteration := transcriptInt(output, "iteration")
	maxIterations := transcriptInt(output, "max_iterations")
	return fmt.Sprintf("工作流进入循环：%s 第 %d/%d 轮 -> %s", node.NodeID, iteration, maxIterations, nextID)
}

func (b *runTranscriptBuilder) appendWorkflowToolNode(node taskdefs.RunNodeResult) {
	toolInput := transcriptNestedRecord(node.Input, "tool")
	toolName := transcriptString(toolInput, "tool_name")
	arguments := toolInput["arguments"]
	output := transcriptFormatValue(transcriptRecord(node.Output)["output"])
	sender := fmt.Sprintf("节点 %s（tool）", node.NodeID)
	b.addEvent(fmt.Sprintf("节点 %s 使用工具：%s", node.NodeID, toolName))
	b.addToolMessage(runToolMessage{
		sender:    sender,
		toolName:  toolName,
		arguments: arguments,
		output:    output,
		status:    node.Status,
	})
}

func (b *runTranscriptBuilder) appendWorkflowTextNode(node taskdefs.RunNodeResult, label string) {
	output := transcriptRecord(node.Output)
	content := transcriptString(output, "response_preview")
	if content == "" {
		content = transcriptFormatValue(output["output"])
	}
	b.addSpeakerMessage(fmt.Sprintf("节点 %s（%s）", node.NodeID, label), content)
}

func (b *runTranscriptBuilder) appendWorkflowAgentNode(node taskdefs.RunNodeResult) {
	output := transcriptRecord(node.Output)
	sessionID := transcriptString(output, "session_id_output")
	sender := fmt.Sprintf("节点 %s（agent）", node.NodeID)
	if sessionID != "" && b.appendAgentSessionMessages(sessionID, sender) {
		return
	}
	b.addSpeakerMessage(sender, transcriptString(output, "response_preview"))
}
