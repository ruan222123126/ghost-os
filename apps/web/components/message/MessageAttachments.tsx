import type { FC } from 'react';
import { useWebLocale } from '@/lib/i18n/provider';
import type { ChatFileAttachment } from '@/lib/types';
import { formatBytes } from '@/lib/chat-view/tool-details/format';

function attachmentMeta(attachment: ChatFileAttachment, fallbackLabel: string): string {
  return [attachment.mimeType, formatBytes(attachment.bytes)].filter(Boolean).join(' · ') || fallbackLabel;
}

export const MessageAttachments: FC<{ attachments: ChatFileAttachment[] }> = ({ attachments }) => {
  const { copy } = useWebLocale();

  return (
    <div className="attachment-list">
      {attachments.map((attachment) => (
        <div key={attachment.artifactId} className="attachment-card">
          <div className="attachment-main">
            <span className="attachment-name">{attachment.name}</span>
            <span className="attachment-meta">{attachmentMeta(attachment, copy.chat.attachmentDefaultMeta)}</span>
            {attachment.note ? <span className="attachment-note">{attachment.note}</span> : null}
          </div>
          <div className="attachment-actions">
            <a className="attachment-link" href={attachment.downloadUrl} download>
              {copy.chat.attachmentDownload}
            </a>
            <a
              className="attachment-link is-secondary"
              href={`${attachment.downloadUrl}?disposition=inline`}
              target="_blank"
              rel="noreferrer"
            >
              {copy.chat.attachmentOpen}
            </a>
          </div>
        </div>
      ))}
    </div>
  );
};
