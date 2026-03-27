import type { ChatFileAttachment, ChatImage, SessionContentPart } from '@/lib/types';

export function attachmentsFromContent(content?: SessionContentPart[]): ChatFileAttachment[] | undefined {
  if (!content || content.length === 0) {
    return undefined;
  }

  const attachments = content
    .filter((part) => part.type === 'file' && part.file)
    .map((part) => ({
      artifactId: part.file!.artifact_id,
      name: part.file!.name,
      downloadUrl: part.file!.download_url,
      mimeType: part.file!.mime_type,
      bytes: part.file!.bytes,
      sha256: part.file!.sha256,
      sourcePath: part.file!.source_path,
      note: part.file!.note,
    }));

  return attachments.length > 0 ? attachments : undefined;
}

export function imagesFromContent(content?: SessionContentPart[]): ChatImage[] | undefined {
  if (!content || content.length === 0) {
    return undefined;
  }

  const images = content
    .filter((part) => part.type === 'image' && part.image)
    .map((part, index) => ({
      id: buildChatImageID(part.image!, index),
      url: part.image!.url,
      path: part.image!.path,
      mimeType: part.image!.mime_type,
      width: part.image!.width,
      height: part.image!.height,
      bytes: part.image!.bytes,
      sha256: part.image!.sha256,
    }));

  return images.length > 0 ? images : undefined;
}

function buildChatImageID(image: NonNullable<SessionContentPart['image']>, index: number): string {
  return image.sha256?.trim()
    || image.url?.trim()
    || image.path?.trim()
    || `session-image:${index}`;
}
