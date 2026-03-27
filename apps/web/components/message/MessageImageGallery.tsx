import Image from 'next/image';
import type { FC } from 'react';
import type { ChatImage } from '@/lib/types';
import { formatBytes } from './format';

export const MessageImageGallery: FC<{ images: ChatImage[] }> = ({ images }) => (
  <div className="message-image-gallery">
    {images.map((image) => (
      <div key={image.id} className="message-image-card">
        {image.url ? (
          <a
            className="message-image-link"
            href={image.url}
            target="_blank"
            rel="noreferrer"
          >
            <Image
              src={image.url}
              alt={image.name || 'Uploaded image'}
              className="message-image-thumb"
              width={image.width || 168}
              height={image.height || 124}
              unoptimized
            />
          </a>
        ) : (
          <div className="message-image-placeholder">
            <span className="message-image-placeholder-label">Image path only</span>
            <span className="message-image-placeholder-path">{image.path || 'Unavailable image source'}</span>
          </div>
        )}
        {buildImageMeta(image) ? <span className="message-image-meta">{buildImageMeta(image)}</span> : null}
      </div>
    ))}
  </div>
);

function buildImageMeta(image: ChatImage): string {
  return [
    image.mimeType,
    formatDimensions(image),
    formatBytes(image.bytes),
  ].filter(Boolean).join(' · ');
}

function formatDimensions(image: ChatImage): string {
  if (!image.width || !image.height) {
    return '';
  }

  return `${image.width}×${image.height}`;
}
