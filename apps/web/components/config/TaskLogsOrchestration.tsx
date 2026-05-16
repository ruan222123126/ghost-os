interface OrchestrationMemberResult {
  round?: number;
  title?: string;
  status?: string;
  content?: string;
  error?: string;
}

interface OrchestrationPrivateDelivery {
  participant_id?: string;
  content?: string;
}

interface OrchestrationTranscriptEntry {
  round?: number;
  speaker?: string;
  agent_id?: string;
  content?: string;
}

interface OrchestrationDispatchResult {
  round?: number;
  action?: string;
  order?: string;
  instruction?: string;
  participant_ids?: string[];
  owner_visible?: boolean;
  private_deliveries?: OrchestrationPrivateDelivery[];
  private_transcript?: OrchestrationTranscriptEntry[];
}

export interface OrchestrationGroupOutput {
  completed_rounds?: number;
  owner_agent_id?: string;
  owner_session_id?: string;
  member_results: OrchestrationMemberResult[];
  dispatch_results?: OrchestrationDispatchResult[];
}

export function OrchestrationRoundsBlock(props: { output: OrchestrationGroupOutput }) {
  const grouped = new Map<number, OrchestrationMemberResult[]>();
  for (const item of props.output.member_results) {
    const round = item.round ?? 0;
    grouped.set(round, [...(grouped.get(round) ?? []), item]);
  }
  return (
    <div className="rounded-[8px] border border-[#E5E5E5] bg-[#FAFAFA] p-2">
      <p className="mb-2 text-[11px] font-semibold uppercase tracking-wide text-[#737373]">
        rounds: {props.output.completed_rounds ?? 0}
      </p>
      {props.output.owner_agent_id ? <p className="mb-2 text-[12px] text-[#525252]">owner: {props.output.owner_agent_id}</p> : null}
      {props.output.owner_session_id ? <p className="mb-2 text-[12px] text-[#525252]">owner_session: {props.output.owner_session_id}</p> : null}
      {props.output.dispatch_results?.length ? <OrchestrationDispatchList dispatchResults={props.output.dispatch_results} /> : null}
      {[...grouped.entries()].map(([round, items]) => <OrchestrationRound key={round} round={round} items={items} />)}
    </div>
  );
}

function OrchestrationDispatchList(props: { dispatchResults: OrchestrationDispatchResult[] }) {
  return (
    <div className="mb-2 space-y-2">
      {props.dispatchResults.map((dispatch, index) => <OrchestrationDispatchCard key={`dispatch-${index}`} dispatch={dispatch} index={index} />)}
    </div>
  );
}

function OrchestrationDispatchCard(props: { dispatch: OrchestrationDispatchResult; index: number }) {
  const { dispatch, index } = props;
  return (
    <details className="rounded-[8px] border border-[#E5E5E5] bg-white p-2" open>
      <summary className="cursor-pointer list-none text-[12px] text-[#111111]">
        dispatch {dispatch.round ?? index + 1}: {dispatch.action ?? 'unknown'}
        {dispatch.order ? ` [${dispatch.order}]` : ''}
        {dispatch.participant_ids?.length ? ` -> ${dispatch.participant_ids.join(', ')}` : ''}
      </summary>
      <div className="mt-2 space-y-1 text-[12px] text-[#111111]">
        {dispatch.instruction ? <p className="whitespace-pre-wrap">instruction: {dispatch.instruction}</p> : null}
        {dispatch.owner_visible !== undefined ? <p>owner_visible: {String(dispatch.owner_visible)}</p> : null}
        {dispatch.private_deliveries?.map((delivery, indexValue) => (
          <p key={`delivery-${indexValue}`} className="whitespace-pre-wrap">
            private_send: {delivery.participant_id ?? 'unknown'} &lt;- {delivery.content ?? ''}
          </p>
        ))}
        {dispatch.private_transcript?.length ? <OrchestrationTranscriptBlock entries={dispatch.private_transcript} /> : null}
      </div>
    </details>
  );
}

function OrchestrationTranscriptBlock(props: { entries: OrchestrationTranscriptEntry[] }) {
  return (
    <div className="rounded-[8px] bg-[#F5F5F5] p-2">
      <p className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-[#737373]">private transcript</p>
      {props.entries.map((entry, index) => (
        <p key={`transcript-${index}`} className="whitespace-pre-wrap text-[12px] text-[#111111]">
          {formatTranscriptEntry(entry)}
        </p>
      ))}
    </div>
  );
}

function OrchestrationRound(props: { round: number; items: OrchestrationMemberResult[] }) {
  return (
    <div className="mb-2 last:mb-0">
      <p className="text-[11px] font-semibold text-[#525252]">round {props.round}</p>
      {props.items.map((item, index) => (
        <p key={`${props.round}-${index}`} className="text-[12px] text-[#111111]">
          {item.title ?? 'member'} [{item.status ?? 'unknown'}]: {item.error ?? item.content ?? ''}
        </p>
      ))}
    </div>
  );
}

function formatTranscriptEntry(entry: OrchestrationTranscriptEntry): string {
  const speaker = entry.speaker || entry.agent_id || 'member';
  const round = entry.round ? `round ${entry.round} ` : '';
  return `${round}${speaker}: ${entry.content ?? ''}`.trim();
}

export function parseOrchestrationGroupOutput(value: unknown): OrchestrationGroupOutput | undefined {
  if (!value || typeof value !== 'object') {
    return undefined;
  }
  const record = value as Record<string, unknown>;
  if (!Array.isArray(record.member_results)) {
    return undefined;
  }
  return {
    completed_rounds: typeof record.completed_rounds === 'number' ? record.completed_rounds : undefined,
    owner_agent_id: typeof record.owner_agent_id === 'string' ? record.owner_agent_id : undefined,
    owner_session_id: typeof record.owner_session_id === 'string' ? record.owner_session_id : undefined,
    member_results: toObjectArray(record.member_results) as OrchestrationMemberResult[],
    dispatch_results: Array.isArray(record.dispatch_results)
      ? toObjectArray(record.dispatch_results).map((item) => ({
        round: typeof item.round === 'number' ? item.round : undefined,
        action: typeof item.action === 'string' ? item.action : undefined,
        order: typeof item.order === 'string' ? item.order : undefined,
        instruction: typeof item.instruction === 'string' ? item.instruction : undefined,
        participant_ids: Array.isArray(item.participant_ids)
          ? item.participant_ids.filter((value): value is string => typeof value === 'string')
          : undefined,
        owner_visible: typeof item.owner_visible === 'boolean' ? item.owner_visible : undefined,
        private_deliveries: Array.isArray(item.private_deliveries)
          ? toObjectArray(item.private_deliveries) as OrchestrationPrivateDelivery[]
          : undefined,
        private_transcript: Array.isArray(item.private_transcript)
          ? toObjectArray(item.private_transcript) as OrchestrationTranscriptEntry[]
          : undefined,
      }))
      : undefined,
  };
}

function toObjectArray(value: unknown): Array<Record<string, unknown>> {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.filter((item): item is Record<string, unknown> => typeof item === 'object' && item !== null);
}
