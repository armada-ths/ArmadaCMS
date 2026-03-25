const hasUriScheme = (value: string) => /^[a-z][a-z\d+.-]*:/i.test(value);

const normalizeHttpScheme = (value: string) => {
  const schemeMatch = value.match(/^(https?:\/\/)/i);
  if (!schemeMatch) return value;

  const scheme = schemeMatch[1].toLowerCase();
  const withoutScheme = value.replace(/^(?:https?:\/\/)+/i, "");

  return `${scheme}${withoutScheme}`;
};

export const normalizeExternalUrl = (value: unknown): unknown => {
  if (typeof value !== "string") return value;

  const trimmed = value.trim();
  if (!trimmed) return "";

  if (trimmed.startsWith("//")) return `https:${trimmed}`;
  if (/^https?:\/\//i.test(trimmed)) return normalizeHttpScheme(trimmed);
  if (hasUriScheme(trimmed)) return trimmed;

  return `https://${trimmed}`;
};

export const formatExternalUrlInput = (value: unknown): unknown => {
  if (typeof value !== "string") return value;

  const trimmed = value.trim();
  if (!trimmed) return "";

  if (trimmed.startsWith("//")) return trimmed.replace(/^\/+/, "");

  return trimmed.replace(/^(?:https?:\/\/)+/i, "");
};

export const validateExternalUrl = (value?: string) => {
  if (value == null || value.trim() === "") return undefined;

  const normalized = normalizeExternalUrl(value);
  if (typeof normalized !== "string") return "Enter a valid external URL";

  try {
    const parsed = new URL(normalized);
    if (!["http:", "https:"].includes(parsed.protocol)) {
      return "Only http(s) links are allowed";
    }

    return undefined;
  } catch {
    return "Enter a valid external URL";
  }
};
