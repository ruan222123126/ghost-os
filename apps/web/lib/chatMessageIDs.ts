export function nextChatMessageID() {
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`;
}
