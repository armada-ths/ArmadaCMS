/**
 * Checks whether a permission string is satisfied by the given set.
 * Supports the same wildcard formats as the Go backend:
 *   "*"            – grants everything
 *   "resource.*"   – grants all actions on a specific resource
 *   "*.action"     – grants a specific action on every resource
 */
export const hasPerm = (permissions: string[], required: string): boolean => {
  const dotIndex = required.indexOf(".");
  const hasResourceAction = dotIndex !== -1;
  const reqResource = hasResourceAction
    ? required.slice(0, dotIndex)
    : required;
  const reqAction = hasResourceAction ? required.slice(dotIndex + 1) : "";

  return permissions.some((p) => {
    if (p === "*" || p === required) return true;
    if (!hasResourceAction) return false;
    const pDot = p.indexOf(".");
    if (pDot === -1) return false;
    const pResource = p.slice(0, pDot);
    const pAction = p.slice(pDot + 1);
    if (pResource === reqResource && pAction === "*") return true;
    if (pResource === "*" && pAction === reqAction) return true;
    return false;
  });
};
