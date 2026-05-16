import type { ToolChatMessage } from '@/lib/types';
import {
  isErrorStatus,
  normalizeInline,
  normalizeToolName,
  parseRecord,
  readString,
  truncateError,
  type RecordValue,
} from './common';
import { resolveToolCallArgs } from './toolCalls';

const ORCHESTRATION_DISPATCH_TOOL = 'orchestration_dispatch';

export function buildOrchestrationDispatchCompact(tool: ToolChatMessage): string {
  const payloads = collectOrchestrationDispatchPayloads(tool);
  if (!isOrchestrationDispatchTool(tool, payloads)) {
    return '';
  }
  if (isErrorStatus(tool.toolStatus)) {
    return '';
  }
  return collectOrchestrationDispatchLines(payloads).join('\n');
}

function isOrchestrationDispatchTool(tool: ToolChatMessage, payloads: RecordValue[]): boolean {
  return normalizeToolName(tool.toolName) === ORCHESTRATION_DISPATCH_TOOL || payloads.length > 0;
}

function collectOrchestrationDispatchPayloads(tool: ToolChatMessage): RecordValue[] {
  const candidates = [
    parseRecord(tool.rawOutput),
    parseRecord(tool.content),
    parseRecord(tool.toolInput),
    resolveToolCallArgs(tool),
  ];
  const payloads: RecordValue[] = [];
  for (const candidate of candidates) {
    payloads.push(...extractOrchestrationDispatchPayloads(candidate));
  }
  return payloads;
}

function extractOrchestrationDispatchPayloads(candidate?: RecordValue): RecordValue[] {
  if (!candidate) {
    return [];
  }
  if (isDispatchPayload(candidate)) {
    return [candidate];
  }
  if (normalizeToolName(readString(candidate, 'tool')) !== ORCHESTRATION_DISPATCH_TOOL) {
    return [];
  }
  const outputPayload = parseRecord(readString(candidate, 'output'));
  if (outputPayload && isDispatchPayload(outputPayload)) {
    return [outputPayload];
  }
  return [];
}

function isDispatchPayload(payload: RecordValue): boolean {
  const action = readString(payload, 'action');
  if (action === 'public_once' || action === 'private_once' || action === 'private_send' || action === 'end_group') {
    return true;
  }
  return Array.isArray(payload.member_results) || Array.isArray(payload.private_transcript) || Array.isArray(payload.private_deliveries) || Array.isArray(payload.private_messages);
}

function collectOrchestrationDispatchLines(payloads: RecordValue[]): string[] {
  const lines: string[] = [];
  const seen = new Set<string>();
  for (const payload of payloads) {
    for (const line of buildDispatchLines(payload)) {
      if (!line || seen.has(line)) {
        continue;
      }
      seen.add(line);
      lines.push(line);
    }
  }
  return lines;
}

function buildDispatchLines(payload: RecordValue): string[] {
  const action = readString(payload, 'action');
  if (action === 'public_once') {
    return buildPublicDispatchLines(payload);
  }
  if (action === 'private_once') {
    return buildPrivateDispatchLines(payload);
  }
  if (action === 'private_send' || (!action && (Array.isArray(payload.private_deliveries) || Array.isArray(payload.private_messages)))) {
    return readDispatchDeliveries(payload, 'private_deliveries').concat(readDispatchDeliveries(payload, 'private_messages'));
  }
  if (action === 'end_group') {
    return ['结束群组'];
  }
  return [];
}

function buildPublicDispatchLines(payload: RecordValue): string[] {
  const speakers = readMemberSpeechLines(payload);
  if (speakers.length > 0) {
    return speakers;
  }
  return buildDispatchHeader('公开轮', payload);
}

function buildPrivateDispatchLines(payload: RecordValue): string[] {
  const transcript = readTranscriptLines(payload);
  const header = buildDispatchHeader('私密子回合', payload);
  return transcript.length > 0 ? header.concat(transcript) : header;
}

function buildDispatchHeader(prefix: string, payload: RecordValue): string[] {
  const participants = readParticipantIDs(payload);
  const instruction = normalizeInline(readString(payload, 'instruction'));
  const lines = participants ? [`${prefix}：${participants}`] : [prefix];
  if (instruction) {
    lines.push(`指令：${truncateError(instruction)}`);
  }
  return lines;
}

function readMemberSpeechLines(payload: RecordValue): string[] {
  const value = payload.member_results;
  if (!Array.isArray(value)) {
    return [];
  }
  return value.flatMap((item) => {
    if (!item || typeof item !== 'object' || Array.isArray(item)) {
      return [];
    }
    const record = item as RecordValue;
    const content = normalizeInline(readString(record, 'content') || readString(record, 'preview') || readString(record, 'error'));
    const speaker = formatSpeaker(readString(record, 'title'), readString(record, 'agent_id'));
    return content && speaker ? [`${speaker}：${truncateError(content)}`] : [];
  });
}

function readTranscriptLines(payload: RecordValue): string[] {
  const value = payload.private_transcript;
  if (!Array.isArray(value)) {
    return [];
  }
  return value.flatMap((item) => {
    if (!item || typeof item !== 'object' || Array.isArray(item)) {
      return [];
    }
    const record = item as RecordValue;
    const speaker = formatSpeaker(readString(record, 'speaker'), readString(record, 'agent_id'));
    const content = normalizeInline(readString(record, 'content'));
    if (!speaker || !content || speaker === 'system') {
      return [];
    }
    return [`${speaker}：${truncateError(content)}`];
  });
}

function readParticipantIDs(payload: RecordValue): string {
  const value = payload.participant_ids;
  if (!Array.isArray(value)) {
    return '';
  }
  return value
    .filter((item): item is string => typeof item === 'string')
    .map((item) => item.trim())
    .filter(Boolean)
    .join('、');
}

function readDispatchDeliveries(payload: RecordValue, key: string): string[] {
  const value = payload[key];
  if (!Array.isArray(value)) {
    return [];
  }
  const lines: string[] = [];
  for (const item of value) {
    if (!item || typeof item !== 'object' || Array.isArray(item)) {
      continue;
    }
    const participantID = normalizeInline(readString(item as RecordValue, 'participant_id'));
    if (!participantID) {
      continue;
    }
    const content = normalizeInline(readString(item as RecordValue, 'content'));
    lines.push(`向${participantID}发了私信：${content}`);
  }
  return lines;
}

function formatSpeaker(title: string, agentID: string): string {
  if (title && agentID && title !== agentID) {
    return `${title}（${agentID}）`;
  }
  return title || agentID;
}
