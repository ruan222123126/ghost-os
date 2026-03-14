import type { FC } from 'react';
import type { ChatFileAttachment } from '@/lib/types';
import { formatBytes } from './format';

function attachmentMeta(attachment: ChatFileAttachment): string {
  return [attachment.mimeType, formatBytes(attachment.bytes)].filter(Boolean).join(' · ') || 'Attachment';
}

export const MessageAttachments: FC<{ attachments: ChatFileAttachment[] }> = ({ attachments }) => (
  <div className="attachment-list">
    {attachments.map((attachment) => (
      <div key={attachment.artifactId} className="attachment-card">
        <div className="attachment-main">
          <span className="attachment-name">{attachment.name}</span>
          <span className="attachment-meta">{attachmentMeta(attachment)}</span>
          {attachment.note ? <span className="attachment-note">{attachment.note}</span> : null}
        </div>
        <div className="attachment-actions">
          <a className="attachment-link" href={attachment.downloadUrl} download>
            Download
          </a>
          <a
            className="attachment-link is-secondary"
            href={`${attachment.downloadUrl}?disposition=inline`}
            target="_blank"
            rel="noreferrer"
          >
            Open
          </a>
        </div>
      </div>
    ))}
  </div>
);
