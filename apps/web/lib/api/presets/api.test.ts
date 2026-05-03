import {
  activatePreset,
  createPreset,
  deletePreset,
  listPresets,
  updatePreset,
} from './api';
import { fetchMock, installFetchMock, mockFetchJSON } from '@/lib/api.test.helpers';

describe('lib/api/presets/api', () => {
  beforeEach(() => {
    installFetchMock();
  });

  it('listPresets calls GET /api/presets', async () => {
    mockFetchJSON({
      status: 'success',
      payload: [
        {
          id: 'preset-a',
          name: 'Default',
          tool_allowlist: ['script_exec'],
          prompt_refs: {},
        },
      ],
      error: '',
    });

    await expect(listPresets()).resolves.toEqual([
      {
        id: 'preset-a',
        name: 'Default',
        tool_allowlist: ['script_exec'],
        prompt_refs: {
          rule: undefined,
          core_job: undefined,
          memory: undefined,
          context: undefined,
        },
      },
    ]);
    expect(fetchMock).toHaveBeenCalledWith('/api/presets', expect.any(Object));
  });

  it('createPreset calls POST /api/presets', async () => {
    mockFetchJSON({
      status: 'success',
      payload: {
        id: 'preset-a',
        name: 'Default',
        tool_allowlist: ['script_exec'],
        prompt_refs: {},
      },
      error: '',
    }, { status: 201 });

    await createPreset({ name: 'Default', tool_allowlist: ['script_exec'], prompt_refs: {} });

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/presets',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ name: 'Default', tool_allowlist: ['script_exec'], prompt_refs: {} }),
      }),
    );
  });

  it('updatePreset calls PATCH /api/presets/:id', async () => {
    mockFetchJSON({
      status: 'success',
      payload: {
        id: 'preset-a',
        name: 'Updated',
        tool_allowlist: ['sfind'],
        prompt_refs: {},
      },
      error: '',
    });

    await updatePreset('preset-a', { name: 'Updated', tool_allowlist: ['sfind'] });

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/presets/preset-a',
      expect.objectContaining({
        method: 'PATCH',
        body: JSON.stringify({ name: 'Updated', tool_allowlist: ['sfind'] }),
      }),
    );
  });

  it('deletePreset calls DELETE /api/presets/:id', async () => {
    mockFetchJSON({
      status: 'success',
      payload: {
        id: 'preset-a',
        name: 'Default',
        tool_allowlist: [],
        prompt_refs: {},
      },
      error: '',
    });

    await deletePreset('preset-a');

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/presets/preset-a',
      expect.objectContaining({
        method: 'DELETE',
      }),
    );
  });

  it('activatePreset calls PUT /api/presets/:id/activate', async () => {
    mockFetchJSON({
      status: 'success',
      payload: {
        id: 'preset-a',
        name: 'Default',
        tool_allowlist: ['script_exec'],
        prompt_refs: {},
      },
      error: '',
    });

    await activatePreset('preset-a');

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/presets/preset-a/activate',
      expect.objectContaining({
        method: 'PUT',
      }),
    );
  });
});
