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
  const grouped = groupMemberResultsByRound(props.output.member_results);

  return (
    <div className="rounded-[8px] border border-[#E5E5E5] bg-[#FAFAFA] p-2">
      <p className="mb-2 text-[11px] font-semibold uppercase tracking-wide text-[#737373]">
        rounds: {props.output.completed_rounds ?? 0}
      </p>
      {props.output.dispatch_results?.length ? <OrchestrationDispatchList dispatchResults={props.output.dispatch_results} /> : null}
      {grouped.map(([round, items]) => <OrchestrationRound key={round} round={round} items={items} />)}
    </div>
  );
}

function groupMemberResultsByRound(items: OrchestrationMemberResult[]): Array<[number, OrchestrationMemberResult[]]> {
  const grouped = new Map<number, OrchestrationMemberResult[]>();
  for (const item of items) {
    const round = item.round ?? 0;
    grouped.set(round, [...(grouped.get(round) ?? []), item]);
  }
  return [...grouped.entries()];
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
  const transcriptEntries = transcriptEntriesForRound(dispatch.private_transcript, dispatch.round);

  return (
    <details className="rounded-[8px] border border-[#E5E5E5] bg-white p-2" open>
      <summary className="cursor-pointer list-none text-[12px] text-[#111111]">
        {formatDispatchSummary(dispatch, index)}
      </summary>
      <div className="mt-2 space-y-1 text-[12px] text-[#111111]">
        <DispatchInstructionLine instruction={dispatch.instruction} />
        <DispatchOwnerVisibleLine ownerVisible={dispatch.owner_visible} />
        <PrivateDeliveriesList deliveries={dispatch.private_deliveries} />
        <MaybeTranscriptBlock entries={transcriptEntries} />
      </div>
    </details>
  );
}

function formatDispatchSummary(dispatch: OrchestrationDispatchResult, index: number): string {
  const round = dispatch.round ?? index + 1;
  const action = dispatch.action ?? 'unknown';
  const order = dispatch.order ? ` [${dispatch.order}]` : '';
  return `dispatch ${round}: ${action}${order}`;
}

function DispatchInstructionLine(props: { instruction?: string }) {
  if (!props.instruction) {
    return null;
  }
  return <p className="whitespace-pre-wrap">instruction: {props.instruction}</p>;
}

function DispatchOwnerVisibleLine(props: { ownerVisible?: boolean }) {
  if (props.ownerVisible === undefined) {
    return null;
  }
  return <p>owner_visible: {String(props.ownerVisible)}</p>;
}

function PrivateDeliveriesList(props: { deliveries?: OrchestrationPrivateDelivery[] }) {
  if (!props.deliveries?.length) {
    return null;
  }

  return (
    <>
      {props.deliveries.map((delivery, indexValue) => (
        <p key={`delivery-${indexValue}`} className="whitespace-pre-wrap">
          private_send: {delivery.content ?? ''}
        </p>
      ))}
    </>
  );
}

function MaybeTranscriptBlock(props: { entries: OrchestrationTranscriptEntry[] }) {
  if (props.entries.length === 0) {
    return null;
  }
  return <OrchestrationTranscriptBlock entries={props.entries} />;
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
  const speaker = entry.speaker || 'member';
  const round = entry.round ? `round ${entry.round} ` : '';
  return `${round}${speaker}: ${entry.content ?? ''}`.trim();
}

function transcriptEntriesForRound(
  entries: OrchestrationTranscriptEntry[] | undefined,
  round: number | undefined,
): OrchestrationTranscriptEntry[] {
  if (!entries?.length) {
    return [];
  }
  if (typeof round !== 'number' || round <= 0) {
    return entries;
  }
  return entries.filter((entry) => entry.round === round);
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
    completed_rounds: numberField(record, 'completed_rounds'),
    owner_agent_id: stringField(record, 'owner_agent_id'),
    owner_session_id: stringField(record, 'owner_session_id'),
    member_results: toObjectArray(record.member_results) as OrchestrationMemberResult[],
    dispatch_results: parseDispatchResults(record.dispatch_results),
  };
}

function parseDispatchResults(value: unknown): OrchestrationDispatchResult[] | undefined {
  if (!Array.isArray(value)) {
    return undefined;
  }
  return toObjectArray(value).map(parseDispatchResult);
}

function parseDispatchResult(item: Record<string, unknown>): OrchestrationDispatchResult {
  return {
    round: numberField(item, 'round'),
    action: stringField(item, 'action'),
    order: stringField(item, 'order'),
    instruction: stringField(item, 'instruction'),
    participant_ids: stringArrayField(item.participant_ids),
    owner_visible: booleanField(item, 'owner_visible'),
    private_deliveries: objectArrayField<OrchestrationPrivateDelivery>(item.private_deliveries),
    private_transcript: objectArrayField<OrchestrationTranscriptEntry>(item.private_transcript),
  };
}

function stringField(record: Record<string, unknown>, key: string): string | undefined {
  return typeof record[key] === 'string' ? record[key] : undefined;
}

function numberField(record: Record<string, unknown>, key: string): number | undefined {
  return typeof record[key] === 'number' ? record[key] : undefined;
}

function booleanField(record: Record<string, unknown>, key: string): boolean | undefined {
  return typeof record[key] === 'boolean' ? record[key] : undefined;
}

function stringArrayField(value: unknown): string[] | undefined {
  if (!Array.isArray(value)) {
    return undefined;
  }
  return value.filter((item): item is string => typeof item === 'string');
}

function objectArrayField<T extends object>(value: unknown): T[] | undefined {
  if (!Array.isArray(value)) {
    return undefined;
  }
  return toObjectArray(value) as T[];
}

function toObjectArray(value: unknown): Array<Record<string, unknown>> {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.filter((item): item is Record<string, unknown> => typeof item === 'object' && item !== null);
}
