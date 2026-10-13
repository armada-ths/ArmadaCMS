/** Parse decimal route IDs without accepting partial, fractional or unsafe values. */
export const parsePositiveIntegerId = (value?: string): number | null => {
  if (value === undefined || !/^\d+$/.test(value)) return null;
  const id = Number(value);
  return Number.isSafeInteger(id) && id > 0 ? id : null;
};
