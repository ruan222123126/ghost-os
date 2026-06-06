package member

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ghost-os/bridge/orchestration/internal/domain/group"
	"ghost-os/bridge/orchestration/internal/ports"
	"ghost-os/bridge/taskdefs"
)

type Runner struct {
	Executor ports.MemberTurnExecutor
}

func (r Runner) RunMember(
	ctx context.Context,
	req ports.MemberRunRequest,
) (ports.MemberResult, error) {
	if req.MemberNode.Agent == nil {
		return SetupError(req, errors.New("orchestration member agent is not configured")), nil
	}
	if r.Executor == nil {
		return SetupError(req, errors.New("orchestration member runner is not configured")), nil
	}
	message := BuildGroupMemberMessage(req)
	return r.Executor.RunMemberTurn(ctx, ports.MemberTurnRequest{
		Message:          message,
		SessionID:        strings.TrimSpace(req.SessionID),
		TraceID:          strings.TrimSpace(req.TraceID),
		Round:            req.Round,
		AgentID:          strings.TrimSpace(req.MemberNode.ID),
		Title:            strings.TrimSpace(req.MemberNode.Agent.Title),
		RuntimeOverrides: taskdefs.CloneTaskRuntimeOverrides(req.MemberNode.Agent.RuntimeOverrides),
	})
}

func BuildGroupMemberMessage(req ports.MemberRunRequest) string {
	privacyText := "公开"
	responseInstruction := "请继续群聊发言，直接输出你这一轮要说的话。"
	if req.Private {
		privacyText = "私密"
		responseInstruction = "请继续私密回合发言，直接输出你这一轮要说的话。"
	}
	instructionBlock := buildInstructionBlock(req.Instruction)
	privateBlock := formatPrivateMessages(req.PrivateMessages)
	return strings.TrimSpace(fmt.Sprintf(
		"成员角色提示：\n%s\n\n群共享上下文：\n%s\n\n当前可见 group transcript：\n%s\n\n当前可见私聊消息：\n%s\n\n轮次信息：\n当前轮次：%d\n总轮次上限：%d\n发言模式：%s\n回合可见性：%s%s\n\n%s",
		req.MemberNode.Agent.Message,
		req.GroupNode.Group.SharedContext,
		req.TranscriptText,
		privateBlock,
		req.Round,
		req.GroupNode.Group.MaxRounds,
		req.GroupNode.Group.SpeakingMode,
		privacyText,
		instructionBlock,
		responseInstruction,
	))
}

func formatPrivateMessages(messages []group.PrivateMessage) string {
	if len(messages) == 0 {
		return "(none)"
	}
	lines := make([]string, 0, len(messages))
	for _, message := range messages {
		lines = append(lines, strings.TrimSpace(message.Content))
	}
	return strings.Join(lines, "\n")
}

func buildInstructionBlock(instruction string) string {
	if strings.TrimSpace(instruction) == "" {
		return ""
	}
	return "\n\n本轮群主附加指令：\n" + strings.TrimSpace(instruction)
}

func SetupError(req ports.MemberRunRequest, err error) ports.MemberResult {
	title := strings.TrimSpace(req.MemberNode.ID)
	if req.MemberNode.Agent != nil && strings.TrimSpace(req.MemberNode.Agent.Title) != "" {
		title = strings.TrimSpace(req.MemberNode.Agent.Title)
	}
	return ports.MemberResult{
		Round:   req.Round,
		AgentID: strings.TrimSpace(req.MemberNode.ID),
		Title:   title,
		Status:  taskdefs.RunStatusError,
		Preview: err.Error(),
		Error:   err.Error(),
	}
}
