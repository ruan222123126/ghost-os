import type {
  ChatSelectedSkill,
  MobileAssistantPart,
  MobileConversationMessage,
  MobileToolCard,
} from "../mobileTypes";

export function areMobileConversationMessagesEqual(
  left: MobileConversationMessage,
  right: MobileConversationMessage,
): boolean {
  if (left === right) {
    return true;
  }
  return left.id === right.id
    && left.role === right.role
    && left.sessionId === right.sessionId
    && left.text === right.text
    && left.thinking === right.thinking
    && areSelectedSkillsEqual(left.selectedSkill, right.selectedSkill)
    && areAssistantPartsEqual(left.parts, right.parts)
    && areToolCardsEqual(left.tools, right.tools);
}

function areSelectedSkillsEqual(left: ChatSelectedSkill | undefined, right: ChatSelectedSkill | undefined): boolean {
  if (left === right) {
    return true;
  }
  return left?.id === right?.id && left?.name === right?.name;
}

function areAssistantPartsEqual(
  left: MobileAssistantPart[] | undefined,
  right: MobileAssistantPart[] | undefined,
): boolean {
  if (left === right) {
    return true;
  }
  if (!left || !right || left.length !== right.length) {
    return false;
  }

  return left.every((leftPart, index) => {
    const rightPart = right[index];
    if (!rightPart || leftPart.id !== rightPart.id || leftPart.kind !== rightPart.kind) {
      return false;
    }
    if (leftPart.kind === "text" && rightPart.kind === "text") {
      return leftPart.text === rightPart.text;
    }
    if (leftPart.kind === "tool" && rightPart.kind === "tool") {
      return areToolCardEqual(leftPart.tool, rightPart.tool);
    }
    return false;
  });
}

function areToolCardsEqual(left: MobileToolCard[] | undefined, right: MobileToolCard[] | undefined): boolean {
  if (left === right) {
    return true;
  }
  if (!left || !right || left.length !== right.length) {
    return false;
  }

  return left.every((leftTool, index) => {
    const rightTool = right[index];
    return Boolean(rightTool) && areToolCardEqual(leftTool, rightTool);
  });
}

function areToolCardEqual(left: MobileToolCard, right: MobileToolCard): boolean {
  return left.id === right.id
    && left.toolName === right.toolName
    && left.toolCallId === right.toolCallId
    && left.status === right.status
    && left.input === right.input
    && left.output === right.output
    && left.error === right.error
    && left.traceId === right.traceId
    && left.approvalDecision === right.approvalDecision
    && left.approvalId === right.approvalId
    && left.approvalInFlight === right.approvalInFlight
    && left.approvalKind === right.approvalKind
    && left.approvalPrompt === right.approvalPrompt
    && left.approvalPayload === right.approvalPayload;
}
