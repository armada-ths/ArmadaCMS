/**
 * Converts a UTC ISO string (from backend) to a local input value
 * compatible with <input type="datetime-local">.
 *
 * Example:
 *  "2025-10-12T08:00:00Z" → "2025-10-12T10:00" (for Stockholm)
 */
export const toLocalInputValue = (value?: string): string => {
  if (!value) return "";
  const date = new Date(value);
  const tzOffset = date.getTimezoneOffset() * 60000;
  return new Date(date.getTime() - tzOffset).toISOString().slice(0, 16);
};

/**
 * Converts a local datetime-local input value back to a UTC ISO string.
 *
 * Example:
 *  "2025-10-12T10:00" → "2025-10-12T08:00:00.000Z"
 */
export const toUTCISOString = (value?: string | null): string | null => {
  if (!value) return null;
  return new Date(value).toISOString();
};
