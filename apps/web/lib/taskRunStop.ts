import type { TaskRunLog } from '@/lib/types';

export interface TaskRunStopRequest {
  run_id: string;
}

export interface TaskRunStopResponse {
  status: 'stopped' | 'not_running';
  message: string;
  task_id: string;
  run_id?: string;
  run?: TaskRunLog;
}
