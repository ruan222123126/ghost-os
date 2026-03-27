import type { ChatImage, ChatImageDraft, SessionImageContent } from '@/lib/types';

const BASE64_CHUNK_SIZE = 32768;
const IMAGE_MIME_PREFIX = 'image/';

interface ImageUploadFile {
  arrayBuffer(): Promise<ArrayBuffer>;
  name: string;
  size: number;
  type: string;
}

export async function createChatImageDrafts(files: Iterable<ImageUploadFile>): Promise<ChatImageDraft[]> {
  const drafts: ChatImageDraft[] = [];

  for (const file of files) {
    validateImageFile(file);
    drafts.push(buildChatImageDraft(file, await toDataUrl(file)));
  }

  return drafts;
}

export function draftImagesToChatImages(drafts: ChatImageDraft[]): ChatImage[] | undefined {
  if (drafts.length === 0) {
    return undefined;
  }

  return drafts.map((draft) => ({
    id: draft.id,
    name: draft.name,
    url: draft.content.url,
    mimeType: draft.content.mime_type,
    bytes: draft.content.bytes,
  }));
}

export function draftImagesToSessionImages(drafts: ChatImageDraft[]): SessionImageContent[] | undefined {
  if (drafts.length === 0) {
    return undefined;
  }

  return drafts.map((draft) => ({ ...draft.content }));
}

function validateImageFile(file: ImageUploadFile): void {
  const mimeType = file.type.trim().toLowerCase();
  if (!mimeType.startsWith(IMAGE_MIME_PREFIX)) {
    throw new Error(`${file.name || 'file'} is not an image`);
  }
}

function buildChatImageDraft(file: ImageUploadFile, url: string): ChatImageDraft {
  return {
    id: nextChatImageID(),
    name: file.name.trim() || 'image',
    content: {
      url,
      mime_type: file.type.trim(),
      bytes: file.size,
    },
  };
}

async function toDataUrl(file: ImageUploadFile): Promise<string> {
  const bytes = new Uint8Array(await file.arrayBuffer());
  const encoder = globalThis.btoa;
  if (typeof encoder !== 'function') {
    throw new Error('base64 encoding is not available in this environment');
  }

  return `data:${file.type.trim()};base64,${encodeBase64(bytes, encoder)}`;
}

function encodeBase64(bytes: Uint8Array, encoder: (data: string) => string): string {
  let binary = '';
  for (let index = 0; index < bytes.length; index += BASE64_CHUNK_SIZE) {
    binary += String.fromCharCode(...bytes.subarray(index, index + BASE64_CHUNK_SIZE));
  }
  return encoder(binary);
}

function nextChatImageID(): string {
  return `chat-image:${Date.now()}:${Math.random().toString(16).slice(2)}`;
}
