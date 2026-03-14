'use client';

import type { FormEvent } from 'react';
import { ignorePromise } from '@/lib/errors';
import {
  ProviderEditorState,
  defaultBaseURLForProviderType,
  providerTypeOptions,
} from '@/lib/configProviders';
import type { ProviderConfig } from '@/lib/types';

interface ProviderEditorFormProps {
  editorMode: 'create' | 'edit';
  editor: ProviderEditorState;
  controlsDisabled: boolean;
  saving: boolean;
  onChangeEditor: (patch: Partial<ProviderEditorState>) => void;
  onSelectProviderType: (providerType: ProviderConfig['type']) => void;
  onSubmit: () => Promise<void>;
  onCancelEditing: () => void;
}

export function ProviderEditorForm(props: ProviderEditorFormProps) {
  const {
    editorMode,
    editor,
    controlsDisabled,
    saving,
    onChangeEditor,
    onSelectProviderType,
    onSubmit,
    onCancelEditing,
  } = props;

  return (
    <form className="settings-form" onSubmit={(event) => handleProviderSubmit(event, onSubmit)}>
      <div>
        <h3>{editorMode === 'edit' ? 'Edit Provider' : 'Add Provider'}</h3>
        <p className="section-copy">Provider names must stay unique. Leaving the API key blank preserves any saved key during edits.</p>
      </div>

      <label className="field-label">
        Name
        <input
          value={editor.name}
          disabled={controlsDisabled}
          onChange={(event) => onChangeEditor({ name: event.target.value })}
          className="input mono"
        />
      </label>

      <label className="field-label">
        Provider Type
        <select
          value={editor.providerType}
          disabled={controlsDisabled}
          onChange={(event) => onSelectProviderType(event.target.value as ProviderConfig['type'])}
          className="select"
        >
          {providerTypeOptions.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
      </label>

      <label className="field-label">
        Base URL
        <input
          value={editor.baseURL}
          disabled={controlsDisabled}
          placeholder={editor.providerType === 'custom'
            ? 'https://example.com/v1'
            : defaultBaseURLForProviderType(editor.providerType)}
          onChange={(event) => onChangeEditor({ baseURL: event.target.value })}
          className="input mono"
        />
      </label>

      <label className="field-label">
        API Key (optional)
        <input
          type="password"
          value={editor.apiKey}
          disabled={controlsDisabled}
          placeholder={editorMode === 'edit' ? 'Leave blank to keep current key' : 'Enter API key'}
          onChange={(event) => onChangeEditor({ apiKey: event.target.value })}
          className="input"
        />
      </label>

      <label className="field-label">
        Models (optional)
        <input
          value={editor.models}
          disabled={controlsDisabled}
          onChange={(event) => onChangeEditor({ models: event.target.value })}
          placeholder="gpt-4o, claude-3-7-sonnet-latest"
          className="input mono"
        />
      </label>

      <div className="settings-actions">
        {editorMode === 'edit' ? (
          <button type="button" onClick={onCancelEditing} className="button-secondary">
            Cancel
          </button>
        ) : null}
        <button type="submit" disabled={controlsDisabled} className="button">
          {saving ? 'Saving...' : editorMode === 'edit' ? 'Update Provider' : 'Create Provider'}
        </button>
      </div>
    </form>
  );
}

function handleProviderSubmit(
  event: FormEvent<HTMLFormElement>,
  onSubmit: () => Promise<void>,
) {
  event.preventDefault();
  ignorePromise(onSubmit());
}
