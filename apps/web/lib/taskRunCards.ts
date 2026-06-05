import type { AgentStreamEvent } from '@/lib/envelope.generated';

export interface TaskRunCard {
  card_id: string;
  run_id?: string;
  kind: string;
  title?: string;
  node_id?: string;
  node_type?: string;
  round?: number;
  iteration?: number;
  branch_id?: string;
  source_session_id?: string;
  started_at?: string;
  status?: string;
  finished_at?: string;
  preview?: string;
  error?: string;
  final_text?: string;
  source_events?: AgentStreamEvent[];
}
