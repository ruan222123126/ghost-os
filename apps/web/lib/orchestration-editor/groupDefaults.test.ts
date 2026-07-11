import { buildDefaultOrchestrationGroupNode, defaultOrchestrationGroupSharedContext } from '@/lib/orchestration-editor/groupDefaults';

describe('lib/orchestration-editor/groupDefaults', () => {
  it('builds localized default group prompts for new orchestration groups', () => {
    expect(defaultOrchestrationGroupSharedContext('zh-CN')).toBe('你们现在在一个群组中，面对着其他人，你们可以跟其他人交流，探讨问题，你们有权保持沉默，也有权说任何话。');
    expect(buildDefaultOrchestrationGroupNode(2, 'zh-CN')).toEqual({
      title: '群组 2',
      shared_context: '你们现在在一个群组中，面对着其他人，你们可以跟其他人交流，探讨问题，你们有权保持沉默，也有权说任何话。',
      speaking_mode: 'sequential',
      owner_agent_id: '',
      max_rounds: 3,
    });
  });
});
