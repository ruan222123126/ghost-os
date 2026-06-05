import type { OrchestrationTaskPayload } from '@/lib/types';

export type OrchestrationView = 'list' | 'create';

export interface OrchestrationState {
  view: OrchestrationView;
  orchestrations: OrchestrationTaskPayload[];
  loading: boolean;
  error: string;
  success: string;
  submitting: boolean;
  name: string;
  runningOrchestrationID: string;
}

export type OrchestrationAction =
  | { type: 'load_start' }
  | { type: 'load_success'; orchestrations: OrchestrationTaskPayload[] }
  | { type: 'load_error'; error: string }
  | { type: 'enter_create' }
  | { type: 'cancel_create' }
  | { type: 'set_name'; name: string }
  | { type: 'create_start' }
  | { type: 'create_success' }
  | { type: 'create_error'; error: string }
  | { type: 'run_start'; id: string }
  | { type: 'run_error'; error: string }
  | { type: 'run_finish' }
  | { type: 'set_success'; success: string }
  | { type: 'set_error'; error: string }
  | { type: 'replace_orchestration'; orchestration: OrchestrationTaskPayload }
  | { type: 'remove_orchestration'; id: string }
  | { type: 'clear_feedback' };

export function createInitialOrchestrationState(): OrchestrationState {
  return {
    view: 'list',
    orchestrations: [],
    loading: true,
    error: '',
    success: '',
    submitting: false,
    name: '',
    runningOrchestrationID: '',
  };
}

export function orchestrationReducer(
  state: OrchestrationState,
  action: OrchestrationAction,
): OrchestrationState {
  switch (action.type) {
    case 'load_start':
      return { ...state, loading: true };
    case 'load_success':
      return { ...state, loading: false, orchestrations: action.orchestrations, error: '' };
    case 'load_error':
      return { ...state, loading: false, error: action.error, success: '' };
    case 'enter_create':
      return { ...state, view: 'create', name: '', error: '', success: '' };
    case 'cancel_create':
      return { ...state, view: 'list', name: '', error: '', success: '' };
    case 'set_name':
      return { ...state, name: action.name };
    case 'create_start':
      return { ...state, submitting: true, error: '', success: '' };
    case 'create_success':
      return { ...state, view: 'list', submitting: false, name: '' };
    case 'create_error':
      return { ...state, submitting: false, error: action.error, success: '' };
    case 'run_start':
      return { ...state, runningOrchestrationID: action.id, error: '', success: '' };
    case 'run_error':
      return { ...state, runningOrchestrationID: '', error: action.error, success: '' };
    case 'run_finish':
      return { ...state, runningOrchestrationID: '' };
    case 'set_success':
      return { ...state, error: '', success: action.success };
    case 'set_error':
      return { ...state, error: action.error, success: '' };
    case 'replace_orchestration':
      return { ...state, orchestrations: replaceOrchestration(state.orchestrations, action.orchestration) };
    case 'remove_orchestration':
      return { ...state, orchestrations: removeOrchestration(state.orchestrations, action.id) };
    case 'clear_feedback':
      return { ...state, error: '', success: '' };
  }
}

export function replaceOrchestration(
  list: OrchestrationTaskPayload[],
  task: OrchestrationTaskPayload,
): OrchestrationTaskPayload[] {
  return list.map((item) => item.id === task.id ? task : item);
}

export function removeOrchestration(
  list: OrchestrationTaskPayload[],
  id: string,
): OrchestrationTaskPayload[] {
  return list.filter((task) => task.id !== id);
}
