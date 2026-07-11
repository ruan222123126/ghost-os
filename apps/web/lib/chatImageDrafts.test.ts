import {
  createChatImageDrafts,
  draftImagesToChatImages,
  draftImagesToSessionImages,
} from './chatImageDrafts';

describe('chatImageDrafts', () => {
  it('creates image drafts from uploaded files', async () => {
    const drafts = await createChatImageDrafts([
      buildImageFile('cat.png', 'image/png', [71, 104, 111, 115, 116]),
    ]);

    expect(drafts).toHaveLength(1);
    expect(drafts[0]).toMatchObject({
      name: 'cat.png',
      content: {
        mime_type: 'image/png',
        bytes: 5,
        url: 'data:image/png;base64,R2hvc3Q=',
      },
    });
    expect(drafts[0].id).toEqual(expect.stringMatching(/^chat-image:/));
  });

  it('rejects non-image files explicitly', async () => {
    await expect(createChatImageDrafts([
      buildImageFile('notes.txt', 'text/plain', [110, 111, 112, 101]),
    ])).rejects.toThrow('notes.txt is not an image');
  });

  it('maps drafts into chat images and request payloads', async () => {
    const drafts = await createChatImageDrafts([
      buildImageFile('cat.png', 'image/png', [71, 104, 111, 115, 116]),
      buildImageFile('dog.jpg', 'image/jpeg', [66, 111, 111]),
    ]);

    expect(draftImagesToChatImages(drafts)).toEqual([
      expect.objectContaining({
        id: drafts[0].id,
        name: 'cat.png',
        url: 'data:image/png;base64,R2hvc3Q=',
      }),
      expect.objectContaining({
        id: drafts[1].id,
        name: 'dog.jpg',
        url: 'data:image/jpeg;base64,Qm9v',
      }),
    ]);
    expect(draftImagesToSessionImages(drafts)).toEqual([
      {
        url: 'data:image/png;base64,R2hvc3Q=',
        mime_type: 'image/png',
        bytes: 5,
      },
      {
        url: 'data:image/jpeg;base64,Qm9v',
        mime_type: 'image/jpeg',
        bytes: 3,
      },
    ]);
  });
});

function buildImageFile(name: string, type: string, bytes: number[]) {
  const buffer = Uint8Array.from(bytes).buffer;

  return {
    arrayBuffer: async () => buffer,
    name,
    size: bytes.length,
    type,
  };
}
