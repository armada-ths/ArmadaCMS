export const hasPerm = (perms: string[], required: string): boolean =>
  perms.some((p) => p === "*" || p === required);
