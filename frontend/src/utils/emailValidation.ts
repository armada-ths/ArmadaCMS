const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export const normalizeEmailInput = (value: unknown): unknown => {
  if (typeof value !== "string") return value;
  return value.trim();
};

export const validateEmailFormat = (value?: string) => {
  if (value == null || value.trim() === "") return undefined;

  return EMAIL_REGEX.test(value.trim())
    ? undefined
    : "Enter a valid email address";
};
