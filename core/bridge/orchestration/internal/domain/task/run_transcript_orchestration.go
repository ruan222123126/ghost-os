package task

import (
	"fmt"
	"strings"

	bridgeTasks "ghost-os/bridge/tasks"
)

func (b *runTranscriptBuilder) appendOrchestrationResults() {
	if len(b.options.Result.NodeResults) == 0 {
		b.addEvent("编排没有产生群组结果")
		return
	}
	for _, node := range b.options.Result.NodeResults {
		b.appendOrchestrationGroupNode(node)
	}
}

func (b *runTranscriptBuilder) appendOrchestrationGroupNode(node bridgeTasks.RunNodeResult) {
	if strings.TrimSpace(node.Error) != "" {
		b.addEvent(fmt.Sprintf("编排群组失败：%s\nerror: %s", node.NodeID, strings.TrimSpace(node.Error)))
	}
	input := transcriptRecord(node.Input)
	title := transcriptString(input, "title")
	if title == "" {
		title = node.NodeID
	}
	b.addEvent(fmt.Sprintf("编排进入群组：%s（%s）", title, node.NodeID))
	output := transcriptRecord(node.Output)
	if dispatches := transcriptSlice(output, "dispatch_results"); len(dispatches) > 0 {
		b.appendOrchestrationDispatches(dispatches)
		return
	}
	b.appendOrchestrationMembers(transcriptSlice(output, "member_results"))
}

func (b *runTranscriptBuilder) appendOrchestrationDispatches(dispatches []any) {
	for _, item := range dispatches {
		dispatch := transcriptRecord(item)
		if dispatch == nil {
			continue
		}
		b.addEvent(orchestrationDispatchEvent(dispatch))
		b.appendOrchestrationMembers(transcriptSlice(dispatch, "member_results"))
	}
}

func orchestrationDispatchEvent(dispatch map[string]any) string {
	round := transcriptInt(dispatch, "round")
	action := transcriptString(dispatch, "action")
	participants := strings.Join(transcriptStringSlice(dispatch["participant_ids"]), ", ")
	instruction := transcriptString(dispatch, "instruction")
	privateDeliveries := len(transcriptSlice(dispatch, "private_deliveries"))
	lines := []string{fmt.Sprintf("编排第 %d 轮调度：%s", round, action)}
	if participants != "" {
		lines = append(lines, "参与者: "+participants)
	}
	if privateDeliveries > 0 {
		lines = append(lines, fmt.Sprintf("私聊投递: %d 条", privateDeliveries))
	}
	if instruction != "" {
		lines = append(lines, "指令: "+instruction)
	}
	return strings.Join(lines, "\n")
}

func (b *runTranscriptBuilder) appendOrchestrationMembers(items []any) {
	lastRound := 0
	for _, item := range items {
		member := transcriptRecord(item)
		if member == nil {
			continue
		}
		round := transcriptInt(member, "round")
		if round != 0 && round != lastRound {
			b.addEvent(fmt.Sprintf("编排进入第 %d 轮", round))
			lastRound = round
		}
		b.appendOrchestrationMember(member)
	}
}

func (b *runTranscriptBuilder) appendOrchestrationMember(member map[string]any) {
	sender := orchestrationMemberSender(member)
	if statusEvent := memberStatusEvent(member); strings.TrimSpace(statusEvent) != "" {
		b.addEvent(statusEvent)
	}
	sessionID := transcriptString(member, "session_id")
	if sessionID != "" && b.appendAgentSessionMessages(sessionID, sender) {
		return
	}
	content := transcriptString(member, "content")
	if content == "" {
		content = transcriptString(member, "preview")
	}
	b.addSpeakerMessage(sender, content)
}

func orchestrationMemberSender(member map[string]any) string {
	title := transcriptString(member, "title")
	agentID := transcriptString(member, "agent_id")
	round := transcriptInt(member, "round")
	if title == "" {
		title = agentID
	}
	if round > 0 {
		return fmt.Sprintf("%s（%s） · 第 %d 轮", title, agentID, round)
	}
	return fmt.Sprintf("%s（%s）", title, agentID)
}

func memberStatusEvent(member map[string]any) string {
	status := transcriptString(member, "status")
	if status == "" || status == runStatusSuccess {
		return ""
	}
	sender := orchestrationMemberSender(member)
	errorText := transcriptString(member, "error")
	if errorText == "" {
		errorText = transcriptString(member, "preview")
	}
	return fmt.Sprintf("成员状态：%s -> %s\n%s", sender, status, errorText)
}
