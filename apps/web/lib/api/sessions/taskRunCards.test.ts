import { parseTaskRunCardFinishedPayload } from './taskRunCards';

describe('lib/api/sessions/taskRunCards', () => {
  it('parses final text from finished run-card events', () => {
    expect(parseTaskRunCardFinishedPayload({
      card_id: 'card-1',
      status: 'success',
      finished_at: '2026-06-06T00:00:00Z',
      preview: 'did: finished',
      final_text: 'did: finished\nnext_step: none',
      source_session_id: 'session-1',
    })).toEqual({
      card_id: 'card-1',
      status: 'success',
      finished_at: '2026-06-06T00:00:00Z',
      preview: 'did: finished',
      error: undefined,
      final_text: 'did: finished\nnext_step: none',
      source_session_id: 'session-1',
    });
  });
});
