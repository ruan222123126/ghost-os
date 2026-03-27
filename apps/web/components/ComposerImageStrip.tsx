// ComposerImageStrip renders selected image previews before the message is sent.

'use client';

import type { FC } from 'react';
import Image from 'next/image';
import type { ChatImageDraft } from '@/lib/types';

interface ComposerImageStripProps {
  images: ChatImageDraft[];
  onRemove: (imageId: string) => void;
}

export const ComposerImageStrip: FC<ComposerImageStripProps> = ({ images, onRemove }) => {
  if (images.length === 0) {
    return null;
  }

  return (
    <div className="composer-image-strip" aria-label="Selected images">
      {images.map((image) => (
        <div key={image.id} className="composer-image-card">
          <Image
            src={image.content.url!}
            alt={image.name}
            className="composer-image-thumb"
            width={112}
            height={84}
            unoptimized
          />
          <div className="composer-image-meta">
            <span className="composer-image-name">{image.name}</span>
            {image.content.bytes ? <span className="composer-image-size">{formatImageSize(image.content.bytes)}</span> : null}
          </div>
          <button
            type="button"
            className="composer-image-remove"
            aria-label={`Remove ${image.name}`}
            onClick={() => onRemove(image.id)}
          >
            <span aria-hidden="true">×</span>
          </button>
        </div>
      ))}
    </div>
  );
};

function formatImageSize(bytes: number): string {
  if (bytes < 1024) {
    return `${bytes} B`;
  }

  return `${(bytes / 1024).toFixed(1)} KB`;
}
