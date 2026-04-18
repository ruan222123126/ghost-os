export const OPEN_TOKEN = '<t:';
export const CLOSE_TOKEN = '</t>';

export function isValidToolTagID(value: string): boolean {
  if (!value) {
    return false;
  }
  for (const char of value) {
    if (char < '0' || char > '9') {
      return false;
    }
  }
  return Number.parseInt(value, 10) > 0;
}
