const SENTENCE_BOUNDARY_PATTERN = /([。！？!?]["')\]}”’）】》」』]*)[ \t]*(?=[\p{L}\p{N}])/gu;

export function formatAssistantTextForDisplay(content: string): string {
  if (!content || content.includes("\n")) {
    return content;
  }

  return content.replace(SENTENCE_BOUNDARY_PATTERN, "$1\n\n");
}
